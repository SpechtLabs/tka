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
	"google.golang.org/protobuf/proto"
)

type GossipClient struct {
	peers []string

	gossipFactor   int
	gossipInterval time.Duration
	store          GossipStore
	listener       *net.Listener
	listenerPort   string
}

type GossipClientOption func(*GossipClient)

func WithGossipFactor(gossipFactor int) GossipClientOption {
	return func(c *GossipClient) { c.gossipFactor = gossipFactor }
}

func WithGossipInterval(gossipInterval time.Duration) GossipClientOption {
	return func(c *GossipClient) { c.gossipInterval = gossipInterval }
}

func WithPeer(peers ...string) GossipClientOption {
	return func(c *GossipClient) { c.peers = append(c.peers, peers...) }
}

func NewGossipClient(store GossipStore, listener *net.Listener, opts ...GossipClientOption) *GossipClient {
	listenerAddr := (*listener).Addr().String()
	_, port, err := net.SplitHostPort(listenerAddr)
	if err != nil {
		otelzap.L().WithError(err).Error("Failed to split listener address")
		return nil
	}

	c := &GossipClient{
		gossipFactor:   3,
		gossipInterval: 1 * time.Second,
		peers:          make([]string, 0),
		store:          store,
		listener:       listener,
		listenerPort:   port,
	}

	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *GossipClient) Start(ctx context.Context) {
	var wg sync.WaitGroup
	wg.Add(2)

	go func() { defer wg.Done(); c.gossipListener(ctx) }()
	go func() { defer wg.Done(); c.gossip(ctx) }()

	wg.Wait()
}

func (c *GossipClient) gossipListener(ctx context.Context) {
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
		go c.handleGossipPeer(ctx, conn)
	}
}

func (c *GossipClient) gossip(ctx context.Context) {
	// startDealy := rand.Int64() % c.gossipInterval.Milliseconds()
	// Sleep for a random amount of time before starting the gossip
	// time.Sleep(time.Millisecond * time.Duration(startDealy))

	// run it one before the ticker starts
	c.gossipSender(ctx)

	// Periodically send gossip messages to the peers until the context is done
	gossipTicker := time.NewTicker(c.gossipInterval)
	for {
		select {
		case <-ctx.Done():
			gossipTicker.Stop()
			return
		case <-gossipTicker.C:
			c.gossipSender(ctx)
		}
	}
}

func (c *GossipClient) gossipSender(ctx context.Context) {
	// copy the peer slice to avoid modifying the original slice
	peers := c.peers

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

	// Gossip with the selected peers
	for _, peer := range peers {
		msg := &messages.GossipMessage{
			Envelope: &messages.GossipMessageEnvelope{
				SrcId:      c.store.GetId(),
				AnswerPort: c.listenerPort,
			},
			Message: &messages.GossipMessage_HeartbeatMessage{
				HeartbeatMessage: &messages.GossipHeartbeatMessage{
					TsUnixNano:       time.Now().UnixNano(),
					VersionMapDigest: c.store.Digest(),
				},
			},
		}

		if err := c.gossipWithPeer(ctx, peer, msg); err != nil {
			fmt.Println("Error gossiping with peer:", err.Display())
		}
	}
}

const varintLenBytes = 10

func (c *GossipClient) gossipWithPeer(ctx context.Context, peer string, msg *messages.GossipMessage) humane.Error {
	// send a gossip message to the peer
	conn, err := net.Dial("tcp", peer)
	if err != nil {
		return humane.Wrap(err, "failed to dial peer")
	}
	defer conn.Close()

	writer := bufio.NewWriter(conn)

	msgBytes, err := proto.Marshal(msg)
	if err != nil {
		return humane.Wrap(err, "failed to marshal gossip message")
	}

	var hdr [varintLenBytes]byte
	hdrLen := binary.PutUvarint(hdr[:], uint64(len(msgBytes)))
	if _, err := writer.Write(hdr[:hdrLen]); err != nil {
		return humane.Wrap(err, "failed to write header")
	}

	if _, err := writer.Write(msgBytes); err != nil {
		return humane.Wrap(err, "failed to write message")
	}

	if err := writer.Flush(); err != nil {
		return humane.Wrap(err, "failed to flush writer")
	}

	return nil
}

func (c *GossipClient) handleGossipPeer(ctx context.Context, conn net.Conn) {
	defer func() { _ = conn.Close() }()

	// read the gossip message from the connection
	reader := bufio.NewReader(conn)
	// read varint length
	hdrLen, err := binary.ReadUvarint(reader)
	if err != nil {
		if err == io.EOF {
			return
		}
		fmt.Printf("Error reading varint length: %v\n", err)
		return
	}

	if hdrLen == 0 {
		// empty message
		return
	}

	// read exactly n bytes
	buf := make([]byte, hdrLen)
	if _, err := io.ReadFull(reader, buf); err != nil {
		if err == io.EOF {
			return
		}
		fmt.Printf("Error reading message: %v\n", err)
		return
	}
	msg := messages.GossipMessage{}
	if err := proto.Unmarshal(buf, &msg); err != nil {
		fmt.Printf("Error unmarshalling message: %v\n", err)
		return
	}

	host, _, err := net.SplitHostPort(conn.RemoteAddr().String())
	if err != nil {
		otelzap.L().WithError(err).Error("Failed to split remote address")
		return
	}

	ip := net.ParseIP(host)
	if ip == nil {
		otelzap.L().WithError(err).Error("Failed to parse IP address")
		return
	}

	// If the IP address is not an IPv4 address, wrap it in square brackets
	if ip.To4() == nil {
		host = fmt.Sprintf("[%s]", host)
	}

	returnAddr := fmt.Sprintf("%s:%s", host, msg.Envelope.AnswerPort)

	// fmt.Println("Received message:", msg.String())

	switch gossipMsgType := msg.Message.(type) {
	case *messages.GossipMessage_HeartbeatMessage:
		gossipMsg := gossipMsgType.HeartbeatMessage
		c.store.Heartbeat(msg.Envelope.SrcId, returnAddr)

		digest := c.store.Digest()
		delta := c.store.Diff(gossipMsg.VersionMapDigest)
		// fmt.Println("Diff:\n", delta.ToString())

		msg := &messages.GossipMessage{
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

		if err := c.gossipWithPeer(ctx, returnAddr, msg); err != nil {
			fmt.Println("Error gossiping with peer:", err.Display())
		}

	case *messages.GossipMessage_GossipDiffMessage:
		gossipMsg := gossipMsgType.GossipDiffMessage
		c.store.Heartbeat(msg.Envelope.SrcId, fmt.Sprintf("%s:%s", host, msg.Envelope.AnswerPort))

		// Generate the delta of the local state and the remote state
		delta := c.store.Diff(gossipMsg.VersionMapDigest)

		// Apply the remote state to the local state
		c.store.Apply(gossipMsg.StateDelta)

		// If there is no delta, we don't need to send anything
		if len(delta) == 0 {
			return
		}
		// Generate the delta message to send to the peer
		msg := &messages.GossipMessage{
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

		if err := c.gossipWithPeer(ctx, returnAddr, msg); err != nil {
			fmt.Println("Error gossiping with peer:", err.Display())
		}

	case *messages.GossipMessage_GossipDeltaMessage:
		gossipMsg := gossipMsgType.GossipDeltaMessage
		c.store.Heartbeat(msg.Envelope.SrcId, fmt.Sprintf("%s:%s", host, msg.Envelope.AnswerPort))
		c.store.Apply(gossipMsg.StateDelta)

	default:
		fmt.Println("Unknown message type:", gossipMsgType)
	}
}
