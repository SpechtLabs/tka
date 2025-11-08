package main

import (
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spechtlabs/tka/pkg/cluster"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(changeIntroducerCmd)

	changeIntroducerCmd.Flags().String("server-addr", "localhost:8080", "The address of the cluster gossip server to join")
	changeIntroducerCmd.Flags().Int("listen-port", 8081, "The port to listen on for incoming gossip messages")
	changeIntroducerCmd.Flags().Duration("gossip-interval", 1*time.Second, "The interval at which to gossip messages to peers")
	changeIntroducerCmd.Flags().Int("gossip-factor", 3, "The factor at which to gossip messages to peers")
	changeIntroducerCmd.Flags().Int("staleness-threshold", 2, "The number of consecutive failed cycles before marking a peer as suspected dead")
	changeIntroducerCmd.Flags().Int("dead-threshold", 4, "The number of consecutive failed cycles before marking a peer as dead and removing it")
	changeIntroducerCmd.Flags().Duration("status-change-interval", 3*time.Second, "The interval at which to change the status of the local node")
}

var changeIntroducerCmd = &cobra.Command{
	Use:   "change-introducer",
	Short: "Join a cluster gossip server and set the local state",
	Long: `The change-introducer command joins a cluster gossip server.
It allows you to join a cluster gossip server and see how it works.
It is not meant to be used in production.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		serverAddr := cmd.Flag("server-addr").Value.String()
		listenPort := cmd.Flag("listen-port").Value.String()
		listenAddr := fmt.Sprintf(":%s", listenPort)
		gossipInterval := cmd.Flag("gossip-interval").Value.String()
		gossipIntervalDuration, err := time.ParseDuration(gossipInterval)
		if err != nil {
			return err
		}
		gossipFactor := cmd.Flag("gossip-factor").Value.String()
		gossipFactorInt, err := strconv.Atoi(gossipFactor)
		if err != nil {
			return err
		}
		stalenessThreshold := cmd.Flag("staleness-threshold").Value.String()
		stalenessThresholdInt, err := strconv.Atoi(stalenessThreshold)
		if err != nil {
			return err
		}
		deadThreshold := cmd.Flag("dead-threshold").Value.String()
		deadThresholdInt, err := strconv.Atoi(deadThreshold)
		if err != nil {
			return err
		}
		store := cluster.NewTestGossipStore[cluster.SerializableString](listenAddr, cluster.WithLocalState(cluster.SerializableString("initial-state")))

		listener, err := net.Listen("tcp", listenAddr)
		if err != nil {
			return err
		}
		defer func() { _ = listener.Close() }()

		gossiper := cluster.NewGossipClient[cluster.SerializableString](
			store,
			&listener,
			cluster.WithGossipFactor[cluster.SerializableString](gossipFactorInt),
			cluster.WithGossipInterval[cluster.SerializableString](gossipIntervalDuration),
			cluster.WithBootstrapPeer[cluster.SerializableString](serverAddr),
			cluster.WithStalenessThreshold[cluster.SerializableString](stalenessThresholdInt),
			cluster.WithDeadThreshold[cluster.SerializableString](deadThresholdInt),
		)

		// Start the gossip client in a goroutine
		go gossiper.Start(cmd.Context())

		statusChangeInterval := cmd.Flag("status-change-interval").Value.String()
		statusChangeIntervalDuration, err := time.ParseDuration(statusChangeInterval)
		if err != nil {
			return err
		}

		// Start the status change goroutine
		go func() {
			for {
				select {
				case <-cmd.Context().Done():
					return
				case <-time.After(statusChangeIntervalDuration):
					store.SetData(cluster.SerializableString(fmt.Sprintf("status-%d", rand.Intn(100))))
				}
			}
		}()

		// Create and start the TUI
		model := newGossipModel(store)
		p := tea.NewProgram(model, tea.WithAltScreen())

		if _, err := p.Run(); err != nil {
			return err
		}

		return nil
	},
}
