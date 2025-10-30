package cluster

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"math/rand/v2"
	"net"
	"slices"
	"sync"
	"time"

	"github.com/sierrasoftworks/humane-errors-go"
	"github.com/spechtlabs/go-otel-utils/otelzap"
	"github.com/spechtlabs/tka/pkg/cluster/messages"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

const varintLenBytes = 10

var (
	tracer = otel.Tracer("github.com/spechtlabs/tka/pkg/cluster")
)

type GossipClient[T SerializableAndStringable] struct {
	peersMu        sync.RWMutex
	bootstrapPeers []string

	gossipFactor   int
	gossipInterval time.Duration
	store          GossipStore[T]
	listener       *net.Listener
	listenerPort   string
}

type GossipClientOption[T SerializableAndStringable] func(*GossipClient[T])

func WithGossipFactor[T SerializableAndStringable](gossipFactor int) GossipClientOption[T] {
	return func(c *GossipClient[T]) { c.gossipFactor = gossipFactor }
}

func WithGossipInterval[T SerializableAndStringable](gossipInterval time.Duration) GossipClientOption[T] {
	return func(c *GossipClient[T]) { c.gossipInterval = gossipInterval }
}

func WithBootstrapPeer[T SerializableAndStringable](peers ...string) GossipClientOption[T] {
	return func(c *GossipClient[T]) {
		c.peersMu.Lock()
		defer c.peersMu.Unlock()
		c.bootstrapPeers = append(c.bootstrapPeers, peers...)
	}
}

func NewGossipClient[T SerializableAndStringable](store GossipStore[T], listener *net.Listener, opts ...GossipClientOption[T]) *GossipClient[T] {
	listenerAddr := (*listener).Addr().String()
	_, port, err := net.SplitHostPort(listenerAddr)
	if err != nil {
		otelzap.L().WithError(err).Error("Failed to split listener address")
		return nil
	}

	c := &GossipClient[T]{
		gossipFactor:   3,
		gossipInterval: 1 * time.Second,
		bootstrapPeers: make([]string, 0),
		store:          store,
		listener:       listener,
		listenerPort:   port,
	}

	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *GossipClient[T]) Start(ctx context.Context) {
	var wg sync.WaitGroup
	wg.Add(2)

	go func() { defer wg.Done(); c.gossipReceiveLoop(ctx) }()
	go func() { defer wg.Done(); c.gossipSendLoop(ctx) }()

	wg.Wait()
}

func (c *GossipClient[T]) gossipReceiveLoop(ctx context.Context) {
	if c.listener == nil {
		otelzap.L().Sugar().Error("Gossip listener not set")
		return
	}

	// Listen for incoming gossip messages
	for {
		conn, err := (*c.listener).Accept()
		if err != nil {
			return
		}
		go func() {
			if err := c.handleGossipPeer(ctx, conn); err != nil {
				otelzap.L().WithError(err).Error("failed to handle gossip peer")
			}
		}()
	}
}

func (c *GossipClient[T]) gossipSendLoop(ctx context.Context) {
	// Sleep for a random amount of time before starting the gossip sender to avoid all nodes gossiping at the same time
	startDealy := rand.Int64() % c.gossipInterval.Milliseconds()
	time.Sleep(time.Millisecond * time.Duration(startDealy))

	// run it one before the ticker starts
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

func (c *GossipClient[T]) gossipToPeers(ctx context.Context) {
	ctx, span := tracer.Start(ctx, "gossip.gossipToPeers",
		trace.WithAttributes(
			attribute.String("gossip.node_id", c.store.GetId()),
			attribute.Int("gossip.factor", c.gossipFactor),
		),
	)
	defer span.End()

	// copy the peer slice to avoid modifying the original slice
	c.peersMu.RLock()
	peers := c.bootstrapPeers
	c.peersMu.RUnlock()

	// Get an up to date list of all our peers and add them to the peers slice if they are not already in it
	for _, peer := range c.store.GetPeers() {
		if slices.Contains(peers, peer.GetAddress()) {
			continue
		}
		peers = append(peers, peer.GetAddress())
	}

	// randomize the peer
	rand.Shuffle(len(peers), func(i, j int) {
		peers[i], peers[j] = peers[j], peers[i]
	})

	// Account for the case where the number of peers is less than the gossip factor
	gossipFactor := min(c.gossipFactor, len(peers))

	// select the first n peers to gossip with where n is the gossip factor
	peers = peers[:gossipFactor]

	span.SetAttributes(
		attribute.Int("gossip.total_peers", len(peers)),
		attribute.StringSlice("gossip.target_peers", peers),
	)

	// Gossip with the selected peers
	var wg sync.WaitGroup
	wg.Add(len(peers))
	for _, peer := range peers {

		digest, errors := c.store.Digest()
		if len(errors) > 0 {
			for _, err := range errors {
				otelzap.L().WithError(err).Ctx(ctx).Error("failed to get digest")
			}

			continue
		}

		msg := &messages.GossipMessage{
			Envelope: &messages.GossipMessageEnvelope{
				SrcId:      c.store.GetId(),
				AnswerPort: c.listenerPort,
			},
			Message: &messages.GossipMessage_HeartbeatMessage{
				HeartbeatMessage: &messages.GossipHeartbeatMessage{
					TsUnixNano:       time.Now().UnixNano(),
					VersionMapDigest: digest,
				},
			},
		}

		go func() {
			defer wg.Done()
			if err := c.gossipWithPeer(ctx, peer, msg); err != nil {
				fmt.Println("Error gossiping with peer:", err.Display())
			}
		}()
	}

	wg.Wait()
}

func (c *GossipClient[T]) gossipWithPeer(ctx context.Context, peer string, msg *messages.GossipMessage) humane.Error {
	ctx, span := tracer.Start(ctx, "gossip.gossipWithPeer",
		trace.WithAttributes(
			attribute.String("gossip.peer", peer),
			attribute.String("gossip.node_id", c.store.GetId()),
			attribute.String("gossip.message_type", fmt.Sprintf("%T", msg.Message)),
		),
	)
	defer span.End()

	// Inject trace context into the message envelope using W3C Trace Context format
	propagator := otel.GetTextMapPropagator()
	carrier := propagation.MapCarrier{}
	propagator.Inject(ctx, carrier)
	if traceparent, ok := carrier["traceparent"]; ok {
		msg.Envelope.Traceparent = traceparent
	}

	// send a gossip message to the peer
	conn, err := net.Dial("tcp", peer)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to dial peer")
		return humane.Wrap(err, "failed to dial peer")
	}
	defer func() { _ = conn.Close() }()

	writer := bufio.NewWriter(conn)

	msgBytes, err := proto.Marshal(msg)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to marshal gossip message")
		return humane.Wrap(err, "failed to marshal gossip message")
	}

	span.SetAttributes(attribute.Int("gossip.message_size_bytes", len(msgBytes)))

	var hdr [varintLenBytes]byte
	hdrLen := binary.PutUvarint(hdr[:], uint64(len(msgBytes)))
	if _, err := writer.Write(hdr[:hdrLen]); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to write header")
		return humane.Wrap(err, "failed to write header")
	}

	if _, err := writer.Write(msgBytes); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to write message")
		return humane.Wrap(err, "failed to write message")
	}

	if err := writer.Flush(); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to flush writer")
		return humane.Wrap(err, "failed to flush writer")
	}

	span.SetStatus(codes.Ok, "message sent successfully")
	return nil
}

func (c *GossipClient[T]) handleGossipPeer(ctx context.Context, conn net.Conn) humane.Error {
	defer func() { _ = conn.Close() }()

	// read the gossip message from the connection
	reader := bufio.NewReader(conn)

	msg, herr := readMessage(reader)
	if herr != nil {
		return humane.Wrap(herr, "failed to read gossip message from peer")
	}

	// If we didn't receive anything, but also not an error, well... return early
	if msg == nil {
		return nil
	}

	// Extract trace context from the message envelope
	if msg.Envelope.Traceparent != "" {
		propagator := otel.GetTextMapPropagator()
		carrier := propagation.MapCarrier{"traceparent": msg.Envelope.Traceparent}
		ctx = propagator.Extract(ctx, carrier)
	}

	ctx, span := tracer.Start(ctx, "gossip.handleGossipPeer",
		trace.WithAttributes(
			attribute.String("gossip.node_id", c.store.GetId()),
			attribute.String("gossip.remote_addr", conn.RemoteAddr().String()),
			attribute.String("gossip.src_id", msg.Envelope.SrcId),
			attribute.String("gossip.message_type", fmt.Sprintf("%T", msg.Message)),
		),
	)
	defer span.End()

	returnAddr, herr := extractAnswerAddress(conn, msg.Envelope.AnswerPort)
	if herr != nil {
		span.RecordError(herr)
		span.SetStatus(codes.Error, "failed to extract answer address")
		return humane.Wrap(herr, "failed to extract answer address")
	}

	span.SetAttributes(attribute.String("gossip.return_addr", returnAddr))

	otelzap.Ctx(ctx).Debug("Received message", zap.String("message", msg.String()))

	if err := c.handleMessage(ctx, returnAddr, msg.Envelope, msg); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to handle message")
		return humane.Wrap(err, "failed to handle message")
	}

	span.SetStatus(codes.Ok, "message handled successfully")
	return nil
}

func (c *GossipClient[T]) handleMessage(ctx context.Context, returnAddr string, envelope *messages.GossipMessageEnvelope, msg *messages.GossipMessage) humane.Error {
	switch gossipMsgType := msg.Message.(type) {
	case *messages.GossipMessage_HeartbeatMessage:
		if err := c.handleHeartbeatMessage(ctx, returnAddr, msg.Envelope, gossipMsgType.HeartbeatMessage); err != nil {
			return humane.Wrap(err, "failed to handle heartbeat message")
		}

	case *messages.GossipMessage_GossipDiffMessage:
		if err := c.handleDiffMessage(ctx, returnAddr, msg.Envelope, gossipMsgType.GossipDiffMessage); err != nil {
			return humane.Wrap(err, "failed to handle diff message")
		}

	case *messages.GossipMessage_GossipDeltaMessage:
		if err := c.handleDeltaMessage(ctx, returnAddr, msg.Envelope, gossipMsgType.GossipDeltaMessage); err != nil {
			return humane.Wrap(err, "failed to handle delta message")
		}

	default:
		return humane.New(fmt.Sprintf("unknown message type: %T", gossipMsgType))
	}

	// If we made it here, we successfully handled the message, so return nil
	return nil
}

func (c *GossipClient[T]) handleHeartbeatMessage(ctx context.Context, returnAddr string, envelope *messages.GossipMessageEnvelope, msg *messages.GossipHeartbeatMessage) humane.Error {
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
		return humane.Wrap(herr, "failed to diff store")
	}

	span.SetAttributes(attribute.Int("gossip.delta_size", len(delta)))

	diffMsg := &messages.GossipMessage{
		Envelope: &messages.GossipMessageEnvelope{
			SrcId:      c.store.GetId(),
			AnswerPort: c.listenerPort,
		},
		Message: &messages.GossipMessage_GossipDiffMessage{
			GossipDiffMessage: &messages.GossipDiffMessage{
				StateDelta:       delta,
				VersionMapDigest: digest,
			},
		},
	}

	if err := c.gossipWithPeer(ctx, returnAddr, diffMsg); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to gossip diff message with peer")
		return humane.Wrap(err, "failed to gossip diff message with peer")
	}

	span.SetStatus(codes.Ok, "heartbeat handled and diff sent")
	return nil
}

func (c *GossipClient[T]) handleDiffMessage(ctx context.Context, returnAddr string, envelope *messages.GossipMessageEnvelope, msg *messages.GossipDiffMessage) humane.Error {
	ctx, span := tracer.Start(ctx, "gossip.handleDiffMessage",
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
		return humane.Wrap(herr, "failed to generate delta")
	}

	// If there is no delta, we don't need to send anything
	if len(delta) == 0 {
		span.SetStatus(codes.Ok, "diff handled, no delta to send")
		return nil
	}

	span.SetAttributes(attribute.Int("gossip.outgoing_delta_size", len(delta)))

	// Generate the delta message to send to the peer
	deltaMsg := &messages.GossipMessage{
		Envelope: &messages.GossipMessageEnvelope{
			SrcId:      c.store.GetId(),
			AnswerPort: c.listenerPort,
		},
		Message: &messages.GossipMessage_GossipDeltaMessage{
			GossipDeltaMessage: &messages.GossipDeltaMessage{
				StateDelta: delta,
			},
		},
	}

	if err := c.gossipWithPeer(ctx, returnAddr, deltaMsg); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to gossip delta message with peer")
		return humane.Wrap(err, "failed to gossip delta message with peer")
	}

	span.SetStatus(codes.Ok, "diff handled and delta sent")
	return nil
}

func (c *GossipClient[T]) handleDeltaMessage(ctx context.Context, returnAddr string, envelope *messages.GossipMessageEnvelope, msg *messages.GossipDeltaMessage) humane.Error {
	_, span := tracer.Start(ctx, "gossip.handleDeltaMessage",
		trace.WithAttributes(
			attribute.String("gossip.node_id", c.store.GetId()),
			attribute.String("gossip.src_id", envelope.SrcId),
			attribute.String("gossip.return_addr", returnAddr),
			attribute.Int("gossip.state_delta_size", len(msg.StateDelta)),
		),
	)
	defer span.End()

	c.store.Heartbeat(envelope.SrcId, returnAddr)

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

func readMessage(reader *bufio.Reader) (*messages.GossipMessage, humane.Error) {
	// read varint length
	hdrLen, err := binary.ReadUvarint(reader)
	if err != nil {
		if err == io.EOF {
			return nil, nil
		}

		return nil, humane.Wrap(err, "failed to read varint length")
	}

	// if the header length is 0, return early
	if hdrLen == 0 {
		return nil, nil
	}

	// read exactly n bytes
	buf := make([]byte, hdrLen)
	if _, err := io.ReadFull(reader, buf); err != nil {
		if err == io.EOF {
			return nil, nil
		}
		return nil, humane.Wrap(err, "failed to read message")
	}

	msg := messages.GossipMessage{}
	if err := proto.Unmarshal(buf, &msg); err != nil {
		return nil, humane.Wrap(err, "failed to unmarshal message")
	}

	return &msg, nil
}

func extractAnswerAddress(conn net.Conn, answerPort string) (string, humane.Error) {
	host, _, err := net.SplitHostPort(conn.RemoteAddr().String())
	if err != nil {
		return "", humane.Wrap(err, "failed to split remote address")
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return "", humane.Wrap(err, "failed to parse IP address")
	}

	// If the IP address is not an IPv4 address, wrap it in square brackets
	if ip.To4() == nil {
		host = fmt.Sprintf("[%s]", host)
	}

	returnAddr := fmt.Sprintf("%s:%s", host, answerPort)
	return returnAddr, nil
}
