package cmd

import (
	"fmt"
	"net"
	"os"
	"strconv"

	"github.com/apparentlymart/go-cidr/cidr"
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

	addrConvCmd := &cobra.Command{
		Use:   "addr-conv",
		Short: "addr-conv <CIDR (string)> <n (int)>",
		Long:  "addr-conv <CIDR (string)> <n (int)>. \nGets the nth IP address from the specified CIDR",
		Run: func(cmd *cobra.Command, args []string) {
			var hostCIDR string
			var pos int64

			if len(args) != 2 {
				failAddrConv()
			}
			hostCIDR = args[0]
			pos, err := strconv.ParseInt(args[1], 0, 0)
			if err != nil {
				failAddrConv()
			}

			_, ipnet, errCidr := net.ParseCIDR(hostCIDR)
			if errCidr != nil {
				failAddrConv()
			}
			ip, errPos := cidr.Host(ipnet, int(pos))
			if errPos != nil {
				failAddrConv()
			}

			fmt.Println(ip)
		},
	}

	ipTypeCmd := &cobra.Command{
		Use:   "ip-type",
		Short: "ip-type <IP/Hostname>",
		Long:  "ip-type <IP/Hostname>. \n Parses the argument and returns whether its an IP or DNS name.",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 1 {
				// ideally we should print an error but the original python binary that this is going to
				// replace did not. Keeping the golang binary in sync with python one for now.
				os.Exit(1)
			}
			// 0th param is the binary name
			toParse := args[0]
			checkIPAddress(toParse)
		},
	}

	etcdRaftCheckerCmd := &cobra.Command{
		Use:   "etcd-raft-checker",
		Short: "etcd-raft-checker command",
		Run: func(cmd *cobra.Command, args []string) {
			var commander Commander = ExecCommander{}
			err := checkEndpointStatus(commander)
			if err != nil {
				fmt.Printf("etcd raft index check with exitCode: %s\n", err)
				os.Exit(1)
			}
		},
	}
	rootSvcCmd.AddCommand(v6TestCmd, addrConvCmd, ipTypeCmd, etcdRaftCheckerCmd)
	return rootSvcCmd
}

func failAddrConv() {
	fmt.Println("None")
	os.Exit(0)
}

func checkIPAddress(ip string) {
	if net.ParseIP(ip) == nil {
		// nil means that IP could not be parsed so it must be a hostname i.e. DNS name
		fmt.Println("DNS")
	} else {
		fmt.Println("IP")
	}
}
