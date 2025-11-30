package cluster

import (
	"context"
	"fmt"
	"math/rand/v2"
	"net"
	"slices"
	"sync"
	"time"

	"github.com/sierrasoftworks/humane-errors-go"
	"github.com/spechtlabs/go-otel-utils/otelzap"
	"github.com/spechtlabs/tka/pkg/cluster/messages"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/protobuf/proto"
)

const (
	// GossipReturnAddrHeader is the gRPC metadata header key used to pass the return address
	// for gossip communication between peers.
	GossipReturnAddrHeader = "x-gossip-return-addr"
)

var (
	tracer = otel.Tracer("github.com/spechtlabs/tka/pkg/cluster")
)

type GossipClient[T SerializableAndStringable] struct {
	messages.UnimplementedGossipServiceServer // Embed for forward compatibility

	peersMu        sync.RWMutex
	bootstrapPeers []string

	gossipFactor       int
	gossipInterval     time.Duration
	stalenessThreshold int // Number of consecutive failed cycles before considering a peer as suspected dead
	deadThreshold      int // Number of consecutive failed cycles before considering a peer as dead (requires to be N > stalenessThreshold)
	store              GossipStore[T]
	grpcServer         *grpc.Server
	listenerPort       string
	listenerAddr       string // Full address (host:port) for return address

	// Connection pool for persistent gRPC clients
	connsMu sync.RWMutex
	conns   map[string]*grpc.ClientConn             // Map of peer address -> gRPC client connection
	clients map[string]messages.GossipServiceClient // Map of peer address -> gRPC service client
}

func NewGossipClient[T SerializableAndStringable](store GossipStore[T], listener net.Listener, opts ...GossipClientOption[T]) *GossipClient[T] {
	listenerAddr := listener.Addr().String()
	_, port, err := net.SplitHostPort(listenerAddr)
	if err != nil {
		otelzap.L().WithError(err).Error("Failed to split listener address", zap.String("nodeID", store.GetId()))
		return nil
	}

	c := &GossipClient[T]{
		gossipFactor:       3,
		gossipInterval:     1 * time.Second,
		stalenessThreshold: 5,  // Default: 5 consecutive failed cycles
		deadThreshold:      10, // Default: 10 consecutive failed cycles
		bootstrapPeers:     make([]string, 0),
		store:              store,
		listenerPort:       port,
		listenerAddr:       listenerAddr,
		conns:              make(map[string]*grpc.ClientConn),
		clients:            make(map[string]messages.GossipServiceClient),
	}

	for _, opt := range opts {
		opt(c)
	}

	if c.deadThreshold <= c.stalenessThreshold {
		otelzap.L().Sugar().With("nodeID", c.store.GetId()).Error("deadThreshold must be greater than stalenessThreshold")
		return nil
	}

	// Create gRPC server with OpenTelemetry stats handler for automatic trace context propagation
	c.grpcServer = grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
	)
	messages.RegisterGossipServiceServer(c.grpcServer, c)

	// Start gRPC server in a goroutine
	go func() {
		if err := c.grpcServer.Serve(listener); err != nil {
			otelzap.L().WithError(err).Error("gRPC server failed", zap.String("nodeID", c.store.GetId()))
		}
	}()

	return c
}

func (c *GossipClient[T]) Start(ctx context.Context) {
	c.gossipToPeers(ctx)

	// Periodically send gossip messages to the peers until the context is done
	gossipTicker := time.NewTicker(c.gossipInterval)
	defer gossipTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-gossipTicker.C:
			c.gossipToPeers(ctx)
		}
	}
}

func (c *GossipClient[T]) Stop() {
	if c.grpcServer != nil {
		c.grpcServer.GracefulStop()
	}

	// Close all gRPC client connections
	c.connsMu.Lock()
	defer c.connsMu.Unlock()
	for addr, conn := range c.conns {
		if err := conn.Close(); err != nil {
			otelzap.L().WithError(err).Error("Failed to close gRPC connection", zap.String("nodeID", c.store.GetId()), zap.String("peer", addr))
		}
		delete(c.conns, addr)
		delete(c.clients, addr)
	}
}

func (c *GossipClient[T]) gossipToPeers(ctx context.Context) {
	ctx, span := tracer.Start(ctx, "gossip.gossipToPeers",
		trace.WithAttributes(
			attribute.String("gossip.node_id", c.store.GetId()),
			attribute.Int("gossip.factor", c.gossipFactor),
		),
	)
	defer span.End()

	// Select peers to gossip with
	peerIDs := c.selectPeersToGossip()
	span.SetAttributes(
		attribute.Int("gossip.total_peers", len(peerIDs)),
		attribute.StringSlice("gossip.target_peer_ids", peerIDs),
	)

	// Gossip with the selected peers
	gossipEnvelope := &messages.GossipMessageEnvelope{
		SrcId: c.store.GetId(),
	}

	var wg sync.WaitGroup
	for _, peerID := range peerIDs {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			c.gossipWithPeer(ctx, id, gossipEnvelope)
		}(peerID)
	}

	wg.Wait()

	// After gossiping, remove stale peers that have exceeded the threshold
	c.removeStalePeers(ctx, span)
}

// removeStalePeers removes stale peers that have exceeded the dead threshold and closes their gRPC connections.
func (c *GossipClient[T]) removeStalePeers(ctx context.Context, span trace.Span) {
	// Get peer addresses before removing them so we can close connections
	peerAddresses := make(map[string]string) // peerID -> address
	allPeers := c.store.GetPeers()
	for _, peer := range allPeers {
		peerAddresses[peer.ID()] = peer.GetAddress()
	}

	removedPeers := c.store.RemoveStalePeers(c.deadThreshold)
	if len(removedPeers) > 0 {
		span.SetAttributes(
			attribute.Int("gossip.removed_stale_peers", len(removedPeers)),
			attribute.StringSlice("gossip.removed_peer_ids", removedPeers),
		)
		otelzap.Ctx(ctx).Info("Removed stale peers",
			zap.String("nodeID", c.store.GetId()),
			zap.Int("count", len(removedPeers)),
			zap.Strings("peer_ids", removedPeers),
		)

		// Close gRPC connections for removed peers
		c.closeConnectionsForPeers(removedPeers, peerAddresses)
	}
}

// selectPeersToGossip selects a subset of peers to gossip with based on the gossip factor.
// It returns the selected peer node IDs.
func (c *GossipClient[T]) selectPeersToGossip() []string {
	// Get all known peers from the store (by ID)
	knownPeers := c.store.GetPeers()
	peerIDs := make([]string, 0, len(knownPeers))
	for _, peer := range knownPeers {
		peerIDs = append(peerIDs, peer.ID())
	}

	// Add bootstrap peers that aren't already known
	// For bootstrap peers, we use the address as a temporary ID until we learn their actual ID
	c.peersMu.RLock()
	bootstrapPeers := c.bootstrapPeers
	c.peersMu.RUnlock()

	// Build a set of known addresses and peer IDs to avoid duplicates
	knownAddresses := make(map[string]bool)
	knownPeerIDs := make(map[string]bool)
	for _, peer := range knownPeers {
		knownAddresses[peer.GetAddress()] = true
		knownPeerIDs[peer.ID()] = true
	}

	for _, bootstrapAddr := range bootstrapPeers {
		// Only add bootstrap peer if:
		// 1. We don't already have a peer with this address
		// 2. We don't already have this address as a peer ID (since we add all known peer IDs first)
		if !knownAddresses[bootstrapAddr] && !knownPeerIDs[bootstrapAddr] {
			// Use address as temporary ID for bootstrap peers we haven't connected to yet
			peerIDs = append(peerIDs, bootstrapAddr)
		}
	}

	// Randomize the peer list to avoid always gossiping with the same peers
	rand.Shuffle(len(peerIDs), func(i, j int) {
		peerIDs[i], peerIDs[j] = peerIDs[j], peerIDs[i]
	})

	// Account for the case where the number of peers is less than the gossip factor
	gossipFactor := min(c.gossipFactor, len(peerIDs))

	// Select the first n peers to gossip with where n is the gossip factor
	return peerIDs[:gossipFactor]
}

func (c *GossipClient[T]) gossipWithPeer(ctx context.Context, peerID string, envelope *messages.GossipMessageEnvelope) {
	// Create a handshake-scoped context that will span the entire 3-way handshake
	// This allows us to trace the complete gossip round through the system
	handshakeCtx, handshakeSpan := tracer.Start(ctx, "gossip.handshake",
		trace.WithAttributes(
			attribute.String("gossip.node_id", c.store.GetId()),
			attribute.String("gossip.peer_id", peerID),
		),
	)
	defer handshakeSpan.End()

	// Add a random jitter up to 150ms to avoid all nodes gossiping at the same time
	<-time.After(time.Duration(rand.Uint32N(150)) * time.Millisecond)

	// Resolve peer address from node ID
	peerAddr, err := c.resolvePeerAddress(peerID)
	if err != nil {
		handshakeSpan.RecordError(err)
		handshakeSpan.SetStatus(codes.Error, "failed to resolve peer address")
		otelzap.L().WithError(err).Ctx(handshakeCtx).Error("failed to resolve peer address", zap.String("nodeID", c.store.GetId()), zap.String("peer_id", peerID))
		return
	}

	handshakeSpan.SetAttributes(attribute.String("gossip.peer_addr", peerAddr))

	// Get digest for heartbeat message
	digest, errors := c.store.Digest()
	if len(errors) > 0 {
		var herr = errors[0]
		if len(errors) > 1 {
			for _, err := range errors[1:] {
				herr = humane.Wrap(herr, err.Display())
			}
		}
		handshakeSpan.RecordError(herr)
		handshakeSpan.SetStatus(codes.Error, "failed to get digest")
		for _, err := range errors {
			otelzap.L().WithError(err).Ctx(handshakeCtx).Error("failed to get digest", zap.String("nodeID", c.store.GetId()))
		}
		return
	}

	// Create heartbeat message
	heartbeatMsg := &messages.GossipHeartbeatMessage{
		TsUnixNano:       time.Now().UnixNano(),
		VersionMapDigest: digest,
	}

	// Send the gossip heartbeat using the handshake context
	// This context will be propagated through the entire handshake
	if err := c.sendHeartbeat(handshakeCtx, peerAddr, envelope, heartbeatMsg); err != nil {
		handshakeSpan.RecordError(err)
		handshakeSpan.SetStatus(codes.Error, "handshake failed")
		otelzap.L().WithError(err).Ctx(handshakeCtx).Error("failed to gossip with peer", zap.String("nodeID", c.store.GetId()), zap.String("peer_id", peerID))

		// Increment failure count for this peer
		c.store.IncrementPeerFailure(peerID, c.stalenessThreshold)
		return
	}

	handshakeSpan.SetStatus(codes.Ok, "handshake completed successfully")
}

// resolvePeerAddress resolves a peer address from a node ID.
// If the peerID is a known peer ID, it looks up the peer in the store.
// If the peerID is a bootstrap peer address (not yet known), it returns the address as-is.
func (c *GossipClient[T]) resolvePeerAddress(peerID string) (string, humane.Error) {
	// Try to get peer from store by ID
	peer := c.store.GetPeer(peerID)
	if peer != nil {
		return peer.GetAddress(), nil
	}

	// If not found, check if it's a bootstrap peer address
	c.peersMu.RLock()
	bootstrapPeers := c.bootstrapPeers
	c.peersMu.RUnlock()

	if slices.Contains(bootstrapPeers, peerID) {
		// It's a bootstrap peer address, return it as-is
		return peerID, nil
	}

	// Not found in store or bootstrap peers
	panic(humane.New(fmt.Sprintf("peer not found: %s", peerID), "ensure the peer is known to the store or is a bootstrap peer"))
}

func (c *GossipClient[T]) sendHeartbeat(ctx context.Context, peer string, envelope *messages.GossipMessageEnvelope, heartbeatMsg *messages.GossipHeartbeatMessage) humane.Error {
	// Create a span for the heartbeat operation (child of handshake span)
	ctx, span := tracer.Start(ctx, "gossip.sendHeartbeat",
		trace.WithAttributes(
			attribute.String("gossip.peer", peer),
			attribute.String("gossip.node_id", c.store.GetId()),
		),
	)
	defer span.End()

	// Create request with hash
	req := &messages.GossipHeartbeatRequest{
		Envelope:         envelope,
		HeartbeatMessage: heartbeatMsg,
	}
	req.Hash = c.hashRequest(req)

	// Get or create persistent gRPC client connection
	client, err := c.getOrCreateClient(peer)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get gRPC client")
		return humane.Wrap(err, "failed to get gRPC client")
	}

	// Create metadata with return address
	md := metadata.New(map[string]string{
		GossipReturnAddrHeader: c.listenerAddr,
	})

	// Trace context is automatically injected by otelgrpc StatsHandler
	// Create a timeout context for the RPC call only, derived from the handshake context
	rpcCtx := metadata.NewOutgoingContext(ctx, md)
	rpcCtx, cancel := context.WithTimeout(rpcCtx, c.gossipInterval)
	defer cancel()

	// Call SendHeartbeat with timeout context
	resp, err := client.SendHeartbeat(rpcCtx, req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to send heartbeat")
		return humane.Wrap(err, "failed to send heartbeat")
	}

	// Validate response hash
	if err := c.validateHash(resp); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalid response hash")
		return humane.Wrap(err, "invalid response hash")
	}

	// Handle the diff response using the original handshake context
	// This ensures trace continuity through the entire handshake
	if err := c.processDiffResponse(ctx, peer, resp.Envelope, resp.GossipDiffMessage); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to handle diff message")
		return humane.Wrap(err, "failed to handle diff message")
	}

	span.SetStatus(codes.Ok, "heartbeat sent successfully")
	return nil
}

func (c *GossipClient[T]) sendDiff(ctx context.Context, peer string, envelope *messages.GossipMessageEnvelope, diffMsg *messages.GossipDiffMessage) (*messages.GossipDeltaResponse, error) {
	// Create a span for the diff operation (child of handshake span)
	ctx, span := tracer.Start(ctx, "gossip.sendDiff",
		trace.WithAttributes(
			attribute.String("gossip.peer", peer),
			attribute.String("gossip.node_id", c.store.GetId()),
		),
	)
	defer span.End()

	// Create request with hash
	req := &messages.GossipDiffRequest{
		Envelope:          envelope,
		GossipDiffMessage: diffMsg,
	}
	req.Hash = c.hashRequest(req)

	// Get or create persistent gRPC client connection
	client, err := c.getOrCreateClient(peer)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get gRPC client")
		return nil, humane.Wrap(err, "failed to get gRPC client")
	}

	// Create metadata with return address
	md := metadata.New(map[string]string{
		GossipReturnAddrHeader: c.listenerAddr,
	})

	// Trace context is automatically injected by otelgrpc StatsHandler
	// Create a timeout context for the RPC call only, derived from the handshake context
	rpcCtx := metadata.NewOutgoingContext(ctx, md)
	rpcCtx, cancel := context.WithTimeout(rpcCtx, c.gossipInterval)
	defer cancel()

	// Call SendDiff with timeout context
	resp, err := client.SendDiff(rpcCtx, req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to send diff")
		return nil, humane.Wrap(err, "failed to send diff")
	}

	// Validate response hash if delta exists
	if resp.HasDelta {
		if err := c.validateHash(resp); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "invalid response hash")
			return nil, humane.Wrap(err, "invalid response hash")
		}
	}

	span.SetStatus(codes.Ok, "diff sent successfully")
	return resp, nil
}

// getClientIfValid checks if a valid client exists for the given peer address.
// The caller must hold the appropriate lock (RLock or Lock) before calling this function.
// Returns the client if valid, or nil if not found or invalid.
func (c *GossipClient[T]) getClientIfValid(peerAddr string) messages.GossipServiceClient {
	client, exists := c.clients[peerAddr]
	if !exists {
		return nil
	}

	conn, connExists := c.conns[peerAddr]
	if !connExists {
		return nil
	}

	state := conn.GetState()
	if state == connectivity.Ready || state == connectivity.Idle {
		return client
	}

	return nil
}

// getOrCreateClient gets or creates a persistent gRPC client connection for the given peer address
func (c *GossipClient[T]) getOrCreateClient(peerAddr string) (messages.GossipServiceClient, error) {
	c.connsMu.RLock()
	// Check if we already have a client for this peer
	if client := c.getClientIfValid(peerAddr); client != nil {
		c.connsMu.RUnlock()
		return client, nil
	}
	c.connsMu.RUnlock()

	// Create new connection
	c.connsMu.Lock()
	defer c.connsMu.Unlock()

	// Double-check after acquiring write lock
	if client := c.getClientIfValid(peerAddr); client != nil {
		return client, nil
	}

	// Clean up invalid connection if it exists
	if conn, exists := c.conns[peerAddr]; exists {
		_ = conn.Close()
		delete(c.conns, peerAddr)
		delete(c.clients, peerAddr)
	}

	conn, err := grpc.NewClient(peerAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
	if err != nil {
		return nil, err
	}

	client := messages.NewGossipServiceClient(conn)
	c.conns[peerAddr] = conn
	c.clients[peerAddr] = client

	return client, nil
}

// closeConnectionsForPeers closes gRPC connections for the given peer IDs.
// peerAddresses is a map of peerID -> address that should contain addresses for all peers
// before they were removed from the store.
func (c *GossipClient[T]) closeConnectionsForPeers(peerIDs []string, peerAddresses map[string]string) {
	if len(peerIDs) == 0 {
		return
	}

	// Build a map of peer addresses to close
	addressesToClose := make(map[string]bool)
	for _, peerID := range peerIDs {
		if addr, exists := peerAddresses[peerID]; exists {
			addressesToClose[addr] = true
		}
	}

	// Close connections for removed peers
	c.connsMu.Lock()
	defer c.connsMu.Unlock()

	for addr := range addressesToClose {
		if conn, exists := c.conns[addr]; exists {
			if err := conn.Close(); err != nil {
				otelzap.L().WithError(err).Warn("Failed to close gRPC connection for removed peer",
					zap.String("nodeID", c.store.GetId()),
					zap.String("peer_addr", addr),
				)
			}
			delete(c.conns, addr)
			delete(c.clients, addr)
			otelzap.L().Debug("Closed gRPC connection for removed peer",
				zap.String("nodeID", c.store.GetId()),
				zap.String("peer_addr", addr),
			)
		}
	}
}

// extractReturnAddressFromMetadata extracts the return address from gRPC metadata headers
func (c *GossipClient[T]) extractReturnAddressFromMetadata(ctx context.Context) (string, humane.Error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", humane.New("no metadata in context")
	}

	// First try to get return address from our custom header
	if vals := md.Get(GossipReturnAddrHeader); len(vals) > 0 {
		return vals[0], nil
	}

	// Fallback: extract from peer info
	p, ok := peer.FromContext(ctx)
	if ok && p.Addr != nil {
		return p.Addr.String(), nil
	}

	// Last resort: try to get from authority header
	if vals := md.Get(":authority"); len(vals) > 0 {
		return vals[0], nil
	}

	return "", humane.New("could not extract return address from metadata")
}

// SendHeartbeat implements the GossipServiceServer interface
func (c *GossipClient[T]) SendHeartbeat(ctx context.Context, req *messages.GossipHeartbeatRequest) (*messages.GossipDiffResponse, error) {
	// Extract return address from metadata headers
	returnAddr, herr := c.extractReturnAddressFromMetadata(ctx)
	if herr != nil {
		return nil, herr
	}

	// Trace context is automatically extracted by otelgrpc StatsHandler
	// Create span for handling the heartbeat (child of the automatically created gRPC span)
	ctx, span := tracer.Start(ctx, "gossip.SendHeartbeat",
		trace.WithAttributes(
			attribute.String("gossip.node_id", c.store.GetId()),
			attribute.String("gossip.src_id", req.Envelope.SrcId),
		),
	)
	defer span.End()

	// Validate request hash
	if err := c.validateRequestHash(req); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalid request hash")
		return nil, err
	}

	// Handle heartbeat and generate response
	resp, err := c.handleHeartbeatMessage(ctx, returnAddr, req.Envelope, req.HeartbeatMessage)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to handle heartbeat")
		return nil, err
	}

	// Add hash to response
	resp.Hash = c.hashResponse(resp)

	span.SetStatus(codes.Ok, "heartbeat handled successfully")
	return resp, nil
}

// SendDiff implements the GossipServiceServer interface
func (c *GossipClient[T]) SendDiff(ctx context.Context, req *messages.GossipDiffRequest) (*messages.GossipDeltaResponse, error) {
	// Extract return address from metadata headers
	returnAddr, herr := c.extractReturnAddressFromMetadata(ctx)
	if herr != nil {
		return nil, herr
	}

	// Trace context is automatically extracted by otelgrpc StatsHandler
	// Create span for handling the diff (child of the automatically created gRPC span)
	ctx, span := tracer.Start(ctx, "gossip.SendDiff",
		trace.WithAttributes(
			attribute.String("gossip.node_id", c.store.GetId()),
			attribute.String("gossip.src_id", req.Envelope.SrcId),
		),
	)
	defer span.End()

	// Validate request hash
	if err := c.validateRequestHash(req); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalid request hash")
		return nil, err
	}

	// Handle diff and generate response
	resp, err := c.processDiffRequest(ctx, returnAddr, req.Envelope, req.GossipDiffMessage)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to handle diff")
		return nil, err
	}

	// Add hash to response if delta exists
	if resp.HasDelta {
		resp.Hash = c.hashResponse(resp)
	}

	span.SetStatus(codes.Ok, "diff handled successfully")
	return resp, nil
}

// SendDelta implements the GossipServiceServer interface
func (c *GossipClient[T]) SendDelta(ctx context.Context, req *messages.GossipDeltaRequest) (*messages.GossipEmptyResponse, error) {
	// Extract return address from metadata headers
	returnAddr, herr := c.extractReturnAddressFromMetadata(ctx)
	if herr != nil {
		return nil, herr
	}

	// Trace context is automatically extracted by otelgrpc StatsHandler
	// Create span for handling the delta (child of the automatically created gRPC span)
	ctx, span := tracer.Start(ctx, "gossip.SendDelta",
		trace.WithAttributes(
			attribute.String("gossip.node_id", c.store.GetId()),
			attribute.String("gossip.src_id", req.Envelope.SrcId),
		),
	)
	defer span.End()

	// Validate request hash
	if err := c.validateRequestHash(req); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalid request hash")
		return nil, err
	}

	// Handle delta
	if err := c.processDeltaMessage(ctx, returnAddr, req.Envelope, req.GossipDeltaMessage); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to handle delta")
		return nil, err
	}

	span.SetStatus(codes.Ok, "delta handled successfully")
	return &messages.GossipEmptyResponse{}, nil
}

func (c *GossipClient[T]) handleHeartbeatMessage(ctx context.Context, returnAddr string, envelope *messages.GossipMessageEnvelope, msg *messages.GossipHeartbeatMessage) (*messages.GossipDiffResponse, error) {
	ctx, span := tracer.Start(ctx, "gossip.handleHeartbeatMessage",
		trace.WithAttributes(
			attribute.String("gossip.node_id", c.store.GetId()),
			attribute.String("gossip.src_id", envelope.SrcId),
			attribute.String("gossip.return_addr", returnAddr),
			attribute.Int("gossip.remote_version_map_size", len(msg.VersionMapDigest)),
		),
	)
	defer span.End()

	c.store.Heartbeat(envelope.SrcId, returnAddr)

	// Check if the remote peer thinks we are suspected dead
	if localDigest, exists := msg.VersionMapDigest[c.store.GetId()]; exists {
		if localDigest.PeerState == messages.PeerState_PEER_STATE_SUSPECTED_DEAD {
			otelzap.Ctx(ctx).Info("Remote peer suspects we are dead, announcing we are alive",
				zap.String("nodeID", c.store.GetId()),
				zap.String("remotePeerID", envelope.SrcId),
			)
			// The local node's state will be announced as healthy in the response
		}
	}

	// Process peer state information from remote digest
	// This updates peer states based on what the remote node reports about other peers
	if errors := c.store.ProcessDigestForPeerStates(msg.VersionMapDigest); len(errors) > 0 {
		for _, err := range errors {
			otelzap.L().WithError(err).Ctx(ctx).Warn("Failed to process digest for peer states",
				zap.String("nodeID", c.store.GetId()),
			)
		}
	}

	digest, errors := c.store.Digest()
	if len(errors) > 0 {
		var herr = errors[0]
		if len(errors) > 1 {
			for _, err := range errors[1:] {
				herr = humane.Wrap(herr, err.Display())
			}
		}
		span.RecordError(herr)
		span.SetStatus(codes.Error, "failed to get digest")
		return nil, herr
	}

	delta, errors := c.store.Diff(msg.VersionMapDigest)
	if len(errors) > 0 {
		var herr = errors[0]

		if len(errors) > 1 {
			for _, err := range errors[1:] {
				herr = humane.Wrap(herr, err.Display())
			}
		}

		span.RecordError(herr)
		span.SetStatus(codes.Error, "failed to diff store")
		return nil, herr
	}

	span.SetAttributes(attribute.Int("gossip.delta_size", len(delta)))

	// Create and return diff response
	resp := &messages.GossipDiffResponse{
		Envelope: &messages.GossipMessageEnvelope{
			SrcId: c.store.GetId(),
		},
		GossipDiffMessage: &messages.GossipDiffMessage{
			StateDelta:       delta,
			VersionMapDigest: digest,
		},
	}

	span.SetStatus(codes.Ok, "heartbeat handled and diff response created")
	return resp, nil
}

// processDiffResponse processes a diff response received from SendHeartbeat.
// It applies the diff, generates a new diff, sends it via SendDiff, and handles the delta response.
func (c *GossipClient[T]) processDiffResponse(ctx context.Context, returnAddr string, envelope *messages.GossipMessageEnvelope, msg *messages.GossipDiffMessage) humane.Error {
	ctx, span := tracer.Start(ctx, "gossip.processDiffResponse",
		trace.WithAttributes(
			attribute.String("gossip.node_id", c.store.GetId()),
			attribute.String("gossip.src_id", envelope.SrcId),
			attribute.String("gossip.return_addr", returnAddr),
			attribute.Int("gossip.state_delta_size", len(msg.StateDelta)),
			attribute.Int("gossip.remote_version_map_size", len(msg.VersionMapDigest)),
		),
	)
	defer span.End()

	c.store.Heartbeat(envelope.SrcId, returnAddr)

	// Check if the remote peer thinks we are suspected dead
	if localDigest, exists := msg.VersionMapDigest[c.store.GetId()]; exists {
		if localDigest.PeerState == messages.PeerState_PEER_STATE_SUSPECTED_DEAD {
			otelzap.Ctx(ctx).Info("Remote peer suspects we are dead, announcing we are alive via diff response",
				zap.String("nodeID", c.store.GetId()),
				zap.String("remotePeerID", envelope.SrcId),
			)
		}
	}

	// Process peer state information from remote digest
	// This updates peer states based on what the remote node reports about other peers
	if errors := c.store.ProcessDigestForPeerStates(msg.VersionMapDigest); len(errors) > 0 {
		for _, err := range errors {
			otelzap.L().WithError(err).Ctx(ctx).Warn("Failed to process digest for peer states",
				zap.String("nodeID", c.store.GetId()),
			)
		}
	}

	// Apply the remote state to the local state
	errors := c.store.Apply(msg.StateDelta)
	if len(errors) > 0 {
		var herr = errors[0]
		if len(errors) > 1 {
			for _, err := range errors[1:] {
				herr = humane.Wrap(herr, err.Display())
			}
		}
		span.RecordError(herr)
		span.SetStatus(codes.Error, "failed to apply diff message to local state")
		return humane.Wrap(herr, "failed to apply diff message to local state")
	}

	span.AddEvent("gossip.state_applied")

	// Get our current digest
	digest, errors := c.store.Digest()
	if len(errors) > 0 {
		var herr = errors[0]
		if len(errors) > 1 {
			for _, err := range errors[1:] {
				herr = humane.Wrap(herr, err.Display())
			}
		}
		span.RecordError(herr)
		span.SetStatus(codes.Error, "failed to get digest")
		return humane.Wrap(herr, "failed to get digest")
	}

	// Generate the delta of the local state and the remote state
	// NOTE: We compare against msg.VersionMapDigest which was generated by the remote node
	// before it applied any updates from us. This means:
	// 1. The remote digest may not reflect state we just sent in msg.StateDelta
	// 2. We may generate deltas for state the remote node already has
	// 3. This causes extra round trips but doesn't break correctness
	delta, errors := c.store.Diff(msg.VersionMapDigest)
	if len(errors) > 0 {
		var herr = errors[0]

		if len(errors) > 1 {
			for _, err := range errors[1:] {
				herr = humane.Wrap(herr, err.Display())
			}
		}

		span.RecordError(herr)
		span.SetStatus(codes.Error, "failed to generate delta")
		return humane.Wrap(herr, "failed to generate delta")
	}

	// If there is no delta, we don't need to send anything
	if len(delta) == 0 {
		span.SetStatus(codes.Ok, "diff handled, no delta to send")
		return nil
	}

	span.SetAttributes(attribute.Int("gossip.outgoing_delta_size", len(delta)))

	// Send diff to peer (which will respond with delta if needed)
	diffMsg := &messages.GossipDiffMessage{
		StateDelta:       delta,
		VersionMapDigest: digest,
	}
	diffEnvelope := &messages.GossipMessageEnvelope{
		SrcId: c.store.GetId(),
	}

	deltaResp, err := c.sendDiff(ctx, returnAddr, diffEnvelope, diffMsg)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to send diff message to peer")
		return humane.Wrap(err, "failed to send diff message to peer")
	}

	// Handle delta response if present
	if deltaResp != nil && deltaResp.HasDelta && deltaResp.GossipDeltaMessage != nil {
		if err := c.processDeltaMessage(ctx, returnAddr, deltaResp.Envelope, deltaResp.GossipDeltaMessage); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to handle delta response")
			return humane.Wrap(err, "failed to handle delta response")
		}
	}

	span.SetStatus(codes.Ok, "diff handled and delta processed")
	return nil
}

// processDiffRequest processes a diff request received via SendDiff.
// It applies the diff, generates a delta response, and returns it.
func (c *GossipClient[T]) processDiffRequest(ctx context.Context, returnAddr string, envelope *messages.GossipMessageEnvelope, msg *messages.GossipDiffMessage) (*messages.GossipDeltaResponse, error) {
	ctx, span := tracer.Start(ctx, "gossip.processDiffRequest",
		trace.WithAttributes(
			attribute.String("gossip.node_id", c.store.GetId()),
			attribute.String("gossip.src_id", envelope.SrcId),
			attribute.String("gossip.return_addr", returnAddr),
			attribute.Int("gossip.state_delta_size", len(msg.StateDelta)),
			attribute.Int("gossip.remote_version_map_size", len(msg.VersionMapDigest)),
		),
	)
	defer span.End()

	c.store.Heartbeat(envelope.SrcId, returnAddr)

	// Check if the remote peer thinks we are suspected dead
	if localDigest, exists := msg.VersionMapDigest[c.store.GetId()]; exists {
		if localDigest.PeerState == messages.PeerState_PEER_STATE_SUSPECTED_DEAD {
			otelzap.Ctx(ctx).Info("Remote peer suspects we are dead, announcing we are alive via diff response",
				zap.String("nodeID", c.store.GetId()),
				zap.String("remotePeerID", envelope.SrcId),
			)
		}
	}

	// Process peer state information from remote digest
	// This updates peer states based on what the remote node reports about other peers
	if errors := c.store.ProcessDigestForPeerStates(msg.VersionMapDigest); len(errors) > 0 {
		for _, err := range errors {
			otelzap.L().WithError(err).Ctx(ctx).Warn("Failed to process digest for peer states",
				zap.String("nodeID", c.store.GetId()),
			)
		}
	}

	// Apply the remote state to the local state
	errors := c.store.Apply(msg.StateDelta)
	if len(errors) > 0 {
		var herr = errors[0]
		if len(errors) > 1 {
			for _, err := range errors[1:] {
				herr = humane.Wrap(herr, err.Display())
			}
		}
		span.RecordError(herr)
		span.SetStatus(codes.Error, "failed to apply diff message to local state")
		return nil, humane.Wrap(herr, "failed to apply diff message to local state")
	}

	span.AddEvent("gossip.state_applied")

	// Generate the delta of the local state and the remote state
	delta, errors := c.store.Diff(msg.VersionMapDigest)
	if len(errors) > 0 {
		var herr = errors[0]

		if len(errors) > 1 {
			for _, err := range errors[1:] {
				herr = humane.Wrap(herr, err.Display())
			}
		}

		span.RecordError(herr)
		span.SetStatus(codes.Error, "failed to generate delta")
		return nil, humane.Wrap(herr, "failed to generate delta")
	}

	// If there is no delta, return empty response
	if len(delta) == 0 {
		span.SetStatus(codes.Ok, "diff handled, no delta to send")
		return &messages.GossipDeltaResponse{
			Envelope: &messages.GossipMessageEnvelope{
				SrcId: c.store.GetId(),
			},
			HasDelta: false,
		}, nil
	}

	span.SetAttributes(attribute.Int("gossip.outgoing_delta_size", len(delta)))

	// Return delta response
	resp := &messages.GossipDeltaResponse{
		Envelope: &messages.GossipMessageEnvelope{
			SrcId: c.store.GetId(),
		},
		GossipDeltaMessage: &messages.GossipDeltaMessage{
			StateDelta: delta,
		},
		HasDelta: true,
	}

	span.SetStatus(codes.Ok, "diff handled and delta response created")
	return resp, nil
}

// processDeltaMessage processes a delta message by applying it to the local state.
// It can be called from both delta requests (SendDelta) and delta responses (from SendDiff).
func (c *GossipClient[T]) processDeltaMessage(ctx context.Context, returnAddr string, envelope *messages.GossipMessageEnvelope, msg *messages.GossipDeltaMessage) humane.Error {
	ctx, span := tracer.Start(ctx, "gossip.processDeltaMessage",
		trace.WithAttributes(
			attribute.String("gossip.node_id", c.store.GetId()),
			attribute.String("gossip.src_id", envelope.SrcId),
			attribute.String("gossip.return_addr", returnAddr),
			attribute.Int("gossip.state_delta_size", len(msg.StateDelta)),
		),
	)
	defer span.End()

	c.store.Heartbeat(envelope.SrcId, returnAddr)

	// Check if the sending peer was marked as suspected dead locally
	if peer := c.store.GetPeer(envelope.SrcId); peer != nil {
		if peer.IsSuspectedDead() || peer.IsDead() {
			// Peer is alive and sending us data, resurrect it
			c.store.ResurrectPeer(envelope.SrcId)
			otelzap.Ctx(ctx).Info("Suspected dead peer responded, resurrecting",
				zap.String("nodeID", c.store.GetId()),
				zap.String("peerID", envelope.SrcId),
			)
		}
	}

	// Apply the remote state to the local state
	errors := c.store.Apply(msg.StateDelta)
	if len(errors) > 0 {
		var herr = errors[0]
		if len(errors) > 1 {
			for _, err := range errors[1:] {
				herr = humane.Wrap(herr, err.Display())
			}
		}
		span.RecordError(herr)
		span.SetStatus(codes.Error, "failed to apply delta message to local state")
		return humane.Wrap(herr, "failed to apply delta message to local state")
	}

	span.AddEvent("gossip.state_applied")
	span.SetStatus(codes.Ok, "delta applied successfully")
	return nil
}

// Helper functions for hash validation and address extraction

func (c *GossipClient[T]) hashRequest(req proto.Message) string {
	// Create a copy without hash for hashing
	switch v := req.(type) {
	case *messages.GossipHeartbeatRequest:
		hash := v.Hash
		v.Hash = ""
		result := shaHashString(v.String())
		v.Hash = hash
		return result
	case *messages.GossipDiffRequest:
		hash := v.Hash
		v.Hash = ""
		result := shaHashString(v.String())
		v.Hash = hash
		return result
	case *messages.GossipDeltaRequest:
		hash := v.Hash
		v.Hash = ""
		result := shaHashString(v.String())
		v.Hash = hash
		return result
	default:
		// This should never happen, but return empty hash if it does
		return ""
	}
}

func (c *GossipClient[T]) hashResponse(resp proto.Message) string {
	// Create a copy without hash for hashing
	switch v := resp.(type) {
	case *messages.GossipDiffResponse:
		hash := v.Hash
		v.Hash = ""
		result := shaHashString(v.String())
		v.Hash = hash
		return result
	case *messages.GossipDeltaResponse:
		hash := v.Hash
		v.Hash = ""
		result := shaHashString(v.String())
		v.Hash = hash
		return result
	default:
		// This should never happen, but return empty hash if it does
		return ""
	}
}

func (c *GossipClient[T]) validateRequestHash(req proto.Message) error {
	var hash string
	var msgWithoutHash proto.Message

	switch v := req.(type) {
	case *messages.GossipHeartbeatRequest:
		if v.Hash == "" {
			return humane.New("request hash is empty")
		}
		hash = v.Hash
		// Create a copy for validation
		msgWithoutHash = &messages.GossipHeartbeatRequest{
			Envelope:         v.Envelope,
			HeartbeatMessage: v.HeartbeatMessage,
			Hash:             "",
		}
	case *messages.GossipDiffRequest:
		if v.Hash == "" {
			return humane.New("request hash is empty")
		}
		hash = v.Hash
		msgWithoutHash = &messages.GossipDiffRequest{
			Envelope:          v.Envelope,
			GossipDiffMessage: v.GossipDiffMessage,
			Hash:              "",
		}
	case *messages.GossipDeltaRequest:
		if v.Hash == "" {
			return humane.New("request hash is empty")
		}
		hash = v.Hash
		msgWithoutHash = &messages.GossipDeltaRequest{
			Envelope:           v.Envelope,
			GossipDeltaMessage: v.GossipDeltaMessage,
			Hash:               "",
		}
	default:
		return humane.New(fmt.Sprintf("unknown request type: %T", req))
	}

	var realHash string
	switch v := msgWithoutHash.(type) {
	case *messages.GossipHeartbeatRequest:
		realHash = shaHashString(v.String())
	case *messages.GossipDiffRequest:
		realHash = shaHashString(v.String())
	case *messages.GossipDeltaRequest:
		realHash = shaHashString(v.String())
	default:
		return humane.New(fmt.Sprintf("unknown request type: %T", msgWithoutHash))
	}
	if realHash != hash {
		return humane.New("request hash is invalid")
	}
	return nil
}

func (c *GossipClient[T]) validateHash(resp proto.Message) error {
	var hash string
	var msgWithoutHash proto.Message

	switch v := resp.(type) {
	case *messages.GossipDiffResponse:
		if v.Hash == "" {
			return humane.New("response hash is empty")
		}
		hash = v.Hash
		msgWithoutHash = &messages.GossipDiffResponse{
			Envelope:          v.Envelope,
			GossipDiffMessage: v.GossipDiffMessage,
			Hash:              "",
		}
	case *messages.GossipDeltaResponse:
		if !v.HasDelta || v.Hash == "" {
			return nil // No hash needed if no delta
		}
		hash = v.Hash
		msgWithoutHash = &messages.GossipDeltaResponse{
			Envelope:           v.Envelope,
			GossipDeltaMessage: v.GossipDeltaMessage,
			Hash:               "",
			HasDelta:           v.HasDelta,
		}
	default:
		return humane.New(fmt.Sprintf("unknown response type: %T", resp))
	}

	var realHash string
	switch v := msgWithoutHash.(type) {
	case *messages.GossipDiffResponse:
		realHash = shaHashString(v.String())
	case *messages.GossipDeltaResponse:
		realHash = shaHashString(v.String())
	default:
		return humane.New(fmt.Sprintf("unknown response type: %T", msgWithoutHash))
	}
	if realHash != hash {
		return humane.New("response hash is invalid")
	}
	return nil
}
