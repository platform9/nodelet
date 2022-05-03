package cmd

import (
	"fmt"
    "github.com/platform9/nodelet/nodeletctl/pkg/nodeletctl"
	"github.com/spf13/cobra"
)

var deleteClusterCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a nodelet based management cluster",
	Long:  "Delete a DU-less nodelet based management cluster on which DDU is deployed",

	RunE: func(command *cobra.Command, args []string) error {
		err := nodeletctl.DeleteCluster(clusterBootstrapFile)
		if err != nil {
			fmt.Printf("\nFailed to create nodelet cluster: %s\n", err)
		}
		return err
	},
}

func init() {
	RootCmd.AddCommand(deleteClusterCmd)
	deleteClusterCmd.Flags().StringVar(&clusterBootstrapFile, "config", "/root/nodeletCluster.yaml", "Path to nodelet bootstrap config")
}
