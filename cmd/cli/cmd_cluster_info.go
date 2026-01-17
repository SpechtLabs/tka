package main

import (
	"context"
	"net/http"
	"os"

	"github.com/spechtlabs/tka/internal/cli/pretty_print"
	"github.com/spechtlabs/tka/pkg/service/api"
	"github.com/spechtlabs/tka/pkg/service/models"
	"github.com/spf13/cobra"
)

var cmdClusterInfo = &cobra.Command{
	Use:   "cluster-info",
	Short: "View cluster information",
	Long: `View cluster information.
This command returns the cluster information that TKA exposes to understand the cluster you're connecting to.`,
	Example: `# View cluster information
tka get cluster-info
`,
	Args:      cobra.ExactArgs(0),
	ValidArgs: []string{},
	RunE:      getClusterInfo,
}

func getClusterInfo(_ *cobra.Command, _ []string) error {
	clusterInfo, _, err := doRequestAndDecode[models.TkaClusterInfo](context.Background(), http.MethodGet, api.ClusterInfoApiRoute, nil, http.StatusOK, http.StatusProcessing)
	if err != nil {
		pretty_print.PrintError(err.Cause())
		os.Exit(1)
	}

	pretty_print.PrintInfo("Cluster Information:")
	pretty_print.PrintClusterInfo(clusterInfo)
	return nil
}
