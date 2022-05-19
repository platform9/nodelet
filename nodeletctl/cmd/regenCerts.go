package cmd

import (
	"fmt"

	"github.com/platform9/nodelet/nodeletctl/pkg/nodeletctl"
	"github.com/spf13/cobra"
)

var regenCertsCmd = &cobra.Command{
	Use:   "regen-certs",
	Short: "Scale up/down a nodelet based cluster",
	Long:  "Scale up/down a nodelete based cluster",

	RunE: func(command *cobra.Command, args []string) error {
		err := nodeletctl.RegenClusterCerts(ClusterCfgFile)
		if err != nil {
			fmt.Printf("\nFailed to update nodelet cluster: %s\n", err)
		}
		return err
	},
}

func init() {
	RootCmd.AddCommand(regenCertsCmd)
}
