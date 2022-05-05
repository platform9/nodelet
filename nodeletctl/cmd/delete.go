package cmd

import (
	"fmt"
    "github.com/platform9/nodelet/nodeletctl/pkg/nodeletctl"
	"github.com/spf13/cobra"
)

var deleteClusterCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a nodelet based cluster",
	Long:  "Delete a nodelete based cluster",

	RunE: func(command *cobra.Command, args []string) error {
		err := nodeletctl.DeleteCluster(ClusterCfgFile)
		if err != nil {
			fmt.Printf("\nFailed to create nodelet cluster: %s\n", err)
		}
		return err
	},
}

func init() {
	RootCmd.AddCommand(deleteClusterCmd)
}
