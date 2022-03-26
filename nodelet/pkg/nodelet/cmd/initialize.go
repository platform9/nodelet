package cmd

import (
	"fmt"
	"github.com/platform9/nodelet/nodelet/pkg/pf9kube"
	"github.com/spf13/cobra"
)

func initCommand() *cobra.Command {

	initCmd := &cobra.Command{
		Use:   "init",
		Short: "command to extract out configuration and other utilities",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("init called")
			err := pf9kube.Extract()
			if err != nil {
				fmt.Printf("Failed to extract pf9-kube: %v", err)
				return fmt.Errorf("failed to extract pf9-kube: %v", err)
			}
			return nil
		},
	}

	return initCmd
}
