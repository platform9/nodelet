package cmd

import (
	"fmt"
	"github.com/platform9/nodelet/nodeletctl/pkg/nodeletctl"
	"github.com/spf13/cobra"
)

var upgradeClusterCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade a nodelet based cluster",
	Long:  "Upgrade a DU-less nodelet based management cluster on remote nodes",

	RunE: func(command *cobra.Command, args []string) error {
		err := nodeletctl.UpgradeCluster(ClusterCfgFile)
		if err != nil {
			fmt.Printf("\nFailed to upgrade nodelet cluster: %s\n", err)
		}
		return err
	},
}

func init() {
	RootCmd.AddCommand(upgradeClusterCmd)
}
