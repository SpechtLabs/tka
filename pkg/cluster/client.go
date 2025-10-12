package cluster

import (
	"context"
	"math/rand/v2"
	"time"
)

type GossipClient struct {
	peers []GossipNode

	gossipFactor   int
	gossipInterval time.Duration
}

type GossipClientOption func(*GossipClient)

func WithGossipFactor(gossipFactor int) GossipClientOption {
	return func(c *GossipClient) { c.gossipFactor = gossipFactor }
}

func WithGossipInterval(gossipInterval time.Duration) GossipClientOption {
	return func(c *GossipClient) { c.gossipInterval = gossipInterval }
}

func NewGossipClient(opts ...GossipClientOption) *GossipClient {
	c := &GossipClient{
		gossipFactor:   3,
		gossipInterval: 1 * time.Second,
		peers:          make([]GossipNode, 0),
	}

	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *GossipClient) Start(ctx context.Context) {
	startDealy := rand.Int64() % c.gossipInterval.Milliseconds()

	// Sleep for a random amount of time before starting the gossip
	time.Sleep(time.Millisecond * time.Duration(startDealy))

	for {
		// copy the peer slice to avoid modifying the original slice
		peers := c.peers

		// randomize the peer
		rand.Shuffle(len(peers), func(i, j int) {
			peers[i], peers[j] = peers[j], peers[i]
		})

		// select the first n peers to gossip with where n is the gossip factor
		peers = peers[:c.gossipFactor]

		for _, peer := range peers {
			c.gossipWithPeer(ctx, peer)
		}
	}
}

func (c *GossipClient) gossipWithPeer(ctx context.Context, peer GossipNode) {
	// send a gossip message to the peer
}
