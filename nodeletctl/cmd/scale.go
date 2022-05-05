package cmd

import (
	"fmt"
    "github.com/platform9/nodelet/nodeletctl/pkg/nodeletctl"
	"github.com/spf13/cobra"
)

var scaleClusterCmd = &cobra.Command{
	Use:   "scale",
	Short: "Scale up/down a nodelet based cluster",
	Long:  "Scale up/down a nodelete based cluster",

	RunE: func(command *cobra.Command, args []string) error {
		err := nodeletctl.ScaleCluster(ClusterCfgFile)
		if err != nil {
			fmt.Printf("\nFailed to update nodelet cluster: %s\n", err)
		}
		return err
	},
}

func init() {
	RootCmd.AddCommand(scaleClusterCmd)
}
