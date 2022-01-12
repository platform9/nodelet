package cobras

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/spf13/cobra"

	_ "github.com/olekukonko/tablewriter"
	_ "go.uber.org/zap"
)

type Options interface {
	Complete(cmd *cobra.Command, args []string) error
	Validate() error
	Run(ctx context.Context) error
}

func Run(opts Options) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		err := opts.Complete(cmd, args)
		if err != nil {
			printErrorAndDie(err)
		}

		err = opts.Validate()
		if err != nil {
			printErrorAndDie(err)
		}

		err = opts.Run(cmd.Context())
		if err != nil {
			printErrorAndDie(err)
		}
	}
}

func Context() (ctx context.Context, cancel func()) {
	// trap Ctrl+C and call cancel on the context
	ctx, origCancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		select {
		case <-c:
			cancel()
		case <-ctx.Done():
		}
	}()
	return ctx, func() {
		signal.Stop(c)
		origCancel()
	}
}

func Execute(cmd *cobra.Command) {
	ctx, cancel := Context()
	defer cancel()
	err := cmd.ExecuteContext(ctx)
	if err != nil {
		printErrorAndDie(err)
	}
}

func printErrorAndDie(err error) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}
