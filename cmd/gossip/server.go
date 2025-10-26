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
	rootCmd.AddCommand(serveCmd)

	serveCmd.Flags().Int("listen-port", 8080, "The port to listen on for gossip messages")
	serveCmd.Flags().Duration("gossip-interval", 1*time.Second, "The interval at which to gossip messages to peers")
	serveCmd.Flags().Int("gossip-factor", 3, "The factor at which to gossip messages to peers")
}

var serveCmd = &cobra.Command{
	Use:   "serve <state>",
	Short: "Start a cluster gossip server",
	Long: `The serve command starts a cluster gossip server.
It allows you to play around with gossip protocols and see how they work.
It is not meant to be used in production.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		listenPort := cmd.Flag("listen-port").Value.String()
		listenAddr := fmt.Sprintf(":%s", listenPort)
		listenPortInt, err := strconv.Atoi(listenPort)
		if err != nil {
			return err
		}
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

		store := cluster.NewTestGossipStore[cluster.SerializableString](listenAddr,
			cluster.WithLocalState(cluster.SerializableString(args[0])),
		)

		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", listenPortInt))
		if err != nil {
			return err
		}
		defer listener.Close()

		gossiper := cluster.NewGossipClient[cluster.SerializableString](
			store,
			&listener,
			cluster.WithGossipFactor[cluster.SerializableString](gossipFactorInt),
			cluster.WithGossipInterval[cluster.SerializableString](gossipIntervalDuration),
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
