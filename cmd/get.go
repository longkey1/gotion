package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/longkey1/gotion/internal/gotion"
	"github.com/longkey1/gotion/internal/gotion/config"
	"github.com/longkey1/gotion/internal/notion"
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
	Long:  `Retrieve a Notion page by its ID or URL and display its properties.`,
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

func runGet(ctx context.Context, pageIDOrURL string, opts *getOptions) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return err
	}

	// Extract page ID from URL if needed
	pageID := gotion.ExtractPageID(pageIDOrURL)

	// Create client based on auth type
	client, err := notion.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	// Build options
	var getOpts *notion.GetPageOptions
	if opts.filterProperties != "" {
		filterProps := strings.Split(opts.filterProperties, ",")
		for i := range filterProps {
			filterProps[i] = strings.TrimSpace(filterProps[i])
		}
		getOpts = &notion.GetPageOptions{
			FilterProperties: filterProps,
		}
	}

	// Get page
	result, err := client.GetPage(ctx, pageID, getOpts)
	if err != nil {
		return fmt.Errorf("failed to get page: %w", err)
	}

	// Output based on source
	if result.Source == "mcp" {
		// MCP returns content directly
		fmt.Println(result.Content)
	} else {
		// API returns structured data
		formatter := gotion.NewFormatter(gotion.OutputFormat(opts.format), os.Stdout)
		// Convert to gotion.Page for formatting
		page := &gotion.Page{
			ID:    result.ID,
			URL:   result.URL,
		}
		return formatter.FormatPage(page)
	}

	return nil
}
