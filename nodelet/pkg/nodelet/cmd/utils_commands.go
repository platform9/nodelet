package cmd

import (
	"fmt"

	"github.com/choria-io/go-validator/ipv6"
	"github.com/spf13/cobra"
)

func newAdvancedCommand() *cobra.Command {
	rootSvcCmd := &cobra.Command{
		Use:   "advanced",
		Short: "Commands related to advanced utils used by pf9-kube",
	}

	// TODO(mithil) - this command can be removed once we move all pf9-kube
	// scripts to go.
	v6TestCmd := &cobra.Command{
		Use:   "is-v6",
		Short: "returns true if provided string is an IPv6 address, false otherwise",
		Run: func(cmd *cobra.Command, args []string) {
			for _, addr := range args {
				isIPv6, _ := ipv6.ValidateString(addr)
				fmt.Printf("%v", isIPv6)
			}

		},
	}

	rootSvcCmd.AddCommand(v6TestCmd)
	return rootSvcCmd
}
