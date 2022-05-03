package cmd

import (
	"fmt"
    "github.com/platform9/nodelet/nodeletctl/pkg/nodeletctl"
	"github.com/spf13/cobra"
)

var scaleClusterCmd = &cobra.Command{
	Use:   "scale",
	Short: "Scale up/down a nodelet based management cluster",
	Long:  "Scale up/down a DU-less nodelet based management cluster on which DDU is deployed",

	RunE: func(command *cobra.Command, args []string) error {
		err := nodeletctl.ScaleCluster(clusterBootstrapFile)
		if err != nil {
			fmt.Printf("\nFailed to update nodelet cluster: %s\n", err)
		}
		return err
	},
}

func init() {
	RootCmd.AddCommand(scaleClusterCmd)
	scaleClusterCmd.Flags().StringVar(&clusterBootstrapFile, "config", "/root/nodeletCluster.yaml", "Path to nodelet bootstrap config")
}
