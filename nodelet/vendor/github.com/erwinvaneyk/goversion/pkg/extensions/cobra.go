package extensions

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/erwinvaneyk/cobras"
	"github.com/spf13/cobra"

	"github.com/erwinvaneyk/goversion"
)

const (
	OutputYAML   = "yaml"
	OutputJSON   = "json"
	OutputJSONPP = "jsonpp"
	OutputShort  = "short"
)

var ValidOutputFormats = []string{
	OutputJSON,
	OutputYAML,
	OutputJSONPP,
	// We add 'short' manually in validate to hide it in the flag description.
}

type CmdOptions struct {
	Short              bool
	OutputFormat       string
	ValidOutputFormats []string
	versionGetter      func() goversion.Info
}

func NewDefaultCmdOptions() *CmdOptions {
	return &CmdOptions{
		Short:              false,
		OutputFormat:       OutputYAML,
		ValidOutputFormats: ValidOutputFormats,
		versionGetter:      goversion.Get,
	}
}

func NewCobraCmdWithDefaults() *cobra.Command {
	return NewCobraCmd(NewDefaultCmdOptions())
}

func NewCobraCmd(opts *CmdOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information.",
		Run:   cobras.Run(opts),
	}

	cmd.Flags().BoolVarP(&opts.Short, "short", "s", opts.Short, "If true, print just the version number.")
	cmd.Flags().StringVarP(&opts.OutputFormat, "output", "o", opts.OutputFormat, fmt.Sprintf("Format of the output. Options: [%s]", strings.Join(opts.ValidOutputFormats, ",")))

	return cmd
}

func (o *CmdOptions) Complete(cmd *cobra.Command, args []string) error {
	if o.Short {
		o.OutputFormat = OutputShort
	}
	return nil
}

func (o *CmdOptions) Validate() error {
	var isValidFormat bool
	for _, validFormat := range append(ValidOutputFormats, OutputShort) {
		if validFormat == o.OutputFormat {
			isValidFormat = true
			break
		}
	}
	if !isValidFormat {
		return errors.New("invalid output format: " + o.OutputFormat)
	}
	return nil
}

func (o *CmdOptions) Run(ctx context.Context) error {
	versionInfo := o.versionGetter()
	switch o.OutputFormat {
	case OutputShort:
		fmt.Println(versionInfo.Version)
	case OutputJSON:
		fmt.Println(versionInfo.ToJSON())
	case OutputJSONPP:
		fmt.Println(versionInfo.ToPrettyJSON())
	case OutputYAML:
		fmt.Print(versionInfo.ToYAML())
	}
	return nil
}
