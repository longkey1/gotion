package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/longkey1/gotion/internal/gotion"
	"github.com/longkey1/gotion/internal/gotion/config"
	"github.com/longkey1/gotion/internal/notion"
	"github.com/longkey1/gotion/internal/notion/types"
	"github.com/spf13/cobra"
)

type updateOptions struct {
	file           string
	propertiesOnly bool
	contentOnly    bool
}

var updateOpts = &updateOptions{}

var updateCmd = &cobra.Command{
	Use:   "update <page_id>",
	Short: "Update an existing Notion page",
	Long: `Update an existing Notion page from Markdown or JSON input.

Input can be provided via stdin or --file flag.

Supported input formats:
  - Markdown with YAML frontmatter (properties in frontmatter, content in body)
  - JSON with "properties" and "content" fields
  - Plain Markdown (content only)`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runUpdate(cmd.Context(), args[0], updateOpts)
	},
}

func init() {
	updateCmd.Flags().StringVar(&updateOpts.file, "file", "", "Input file path (default: stdin)")
	updateCmd.Flags().BoolVar(&updateOpts.propertiesOnly, "properties-only", false, "Update properties only")
	updateCmd.Flags().BoolVar(&updateOpts.contentOnly, "content-only", false, "Update content only")

	rootCmd.AddCommand(updateCmd)
}

func runUpdate(ctx context.Context, pageIDOrURL string, opts *updateOptions) error {
	if opts.propertiesOnly && opts.contentOnly {
		return fmt.Errorf("--properties-only and --content-only are mutually exclusive")
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return err
	}

	pageID := gotion.ExtractPageID(pageIDOrURL)

	// Read input
	var input *gotion.ParsedInput
	if opts.file != "" {
		f, err := os.Open(opts.file)
		if err != nil {
			return fmt.Errorf("failed to open file: %w", err)
		}
		defer f.Close()
		input, err = gotion.ParseInput(f)
		if err != nil {
			return fmt.Errorf("failed to parse input: %w", err)
		}
	} else {
		input, err = gotion.ParseInput(os.Stdin)
		if err != nil {
			return fmt.Errorf("failed to parse input: %w", err)
		}
	}

	// Build update options
	updatePageOpts := &types.UpdatePageOptions{}

	if !opts.contentOnly && input.Properties != nil {
		updatePageOpts.Properties = input.Properties
	}

	if !opts.propertiesOnly && input.Content != "" {
		updatePageOpts.Content = &input.Content
	}

	// Create client
	client, err := notion.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	// Update page
	result, err := client.UpdatePage(ctx, pageID, updatePageOpts)
	if err != nil {
		return fmt.Errorf("failed to update page: %w", err)
	}

	fmt.Println(string(result.RawJSON))
	return nil
}
