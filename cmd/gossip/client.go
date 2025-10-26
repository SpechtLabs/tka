package main

import (
	"fmt"
	"net"
	"strconv"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spechtlabs/tka/pkg/cluster"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(clientCmd)

	clientCmd.Flags().String("server-addr", "localhost:8080", "The address of the cluster gossip server to join")
	clientCmd.Flags().Int("listen-port", 8081, "The port to listen on for incoming gossip messages")
	clientCmd.Flags().Duration("gossip-interval", 1*time.Second, "The interval at which to gossip messages to peers")
	clientCmd.Flags().Int("gossip-factor", 3, "The factor at which to gossip messages to peers")
}

var clientCmd = &cobra.Command{
	Use:   "client <state>",
	Short: "Join a cluster gossip server and set the local state",
	Long: `The client command joins a cluster gossip server.
It allows you to join a cluster gossip server and see how it works.
It is not meant to be used in production.`,
	Args: cobra.ExactArgs(1),
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

		store := cluster.NewTestGossipStore(listenAddr,
			cluster.WithLocalState(cluster.SerializableString(args[0])),
		)

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
			cluster.WithPeer[cluster.SerializableString](serverAddr),
		)

		// Start the gossip client in a goroutine
		go gossiper.Start(cmd.Context())

		// Create and start the TUI
		model := newGossipModel(store)
		p := tea.NewProgram(model, tea.WithAltScreen())

		if _, err := p.Run(); err != nil {
			return err
		}

		return nil
	},
}
