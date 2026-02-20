package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/longkey1/gotion/internal/gotion"
	"github.com/longkey1/gotion/internal/gotion/config"
	"github.com/spf13/cobra"
)

type getOptions struct {
	format           string
	filterProperties string
}

var getOpts = &getOptions{}

var getCmd = &cobra.Command{
	Use:   "get <page_id>",
	Short: "Get a single Notion page",
	Long:  `Retrieve a Notion page by its ID and display its properties.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runGet(cmd.Context(), args[0], getOpts)
	},
}

func init() {
	getCmd.Flags().StringVarP(&getOpts.format, "format", "f", "text", "Output format: json, text, table")
	getCmd.Flags().StringVar(&getOpts.filterProperties, "filter-properties", "", "Filter properties to retrieve (comma-separated)")

	rootCmd.AddCommand(getCmd)
}

func runGet(ctx context.Context, pageID string, opts *getOptions) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return err
	}

	client := gotion.NewClient(cfg.Token)

	var filterProps []string
	if opts.filterProperties != "" {
		filterProps = strings.Split(opts.filterProperties, ",")
		for i := range filterProps {
			filterProps[i] = strings.TrimSpace(filterProps[i])
		}
	}

	page, err := client.GetPage(ctx, pageID, filterProps)
	if err != nil {
		return fmt.Errorf("failed to get page: %w", err)
	}

	formatter := gotion.NewFormatter(gotion.OutputFormat(opts.format), os.Stdout)
	return formatter.FormatPage(page)
}
