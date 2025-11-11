package cluster

import "time"

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

func WithStalenessThreshold[T SerializableAndStringable](threshold int) GossipClientOption[T] {
	return func(c *GossipClient[T]) { c.stalenessThreshold = threshold }
}

func WithDeadThreshold[T SerializableAndStringable](threshold int) GossipClientOption[T] {
	return func(c *GossipClient[T]) { c.deadThreshold = threshold }
}
