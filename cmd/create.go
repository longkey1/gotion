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

type createOptions struct {
	parent     string
	parentType string
	title      string
	file       string
}

var createOpts = &createOptions{}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new Notion page",
	Long: `Create a new Notion page from Markdown or JSON input.

Input can be provided via stdin or --file flag.

Supported input formats:
  - Markdown with YAML frontmatter (title and properties in frontmatter)
  - JSON with "properties" and "content" fields
  - Plain Markdown (content only, use --title for the page title)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runCreate(cmd.Context(), createOpts)
	},
}

func init() {
	createCmd.Flags().StringVar(&createOpts.parent, "parent", "", "Parent page or database ID")
	createCmd.Flags().StringVar(&createOpts.parentType, "parent-type", "page_id", "Parent type: page_id, database_id, data_source_id")
	createCmd.Flags().StringVar(&createOpts.title, "title", "", "Page title (overrides input)")
	createCmd.Flags().StringVar(&createOpts.file, "file", "", "Input file path (default: stdin)")

	rootCmd.AddCommand(createCmd)
}

func runCreate(ctx context.Context, opts *createOptions) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return err
	}

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

	// Override title from flag
	if opts.title != "" {
		if input.Properties == nil {
			input.Properties = make(map[string]interface{})
		}
		input.Properties["title"] = opts.title
	}

	// Build create options
	createPageOpts := &types.CreatePageOptions{
		Properties: input.Properties,
		Content:    input.Content,
	}

	if opts.parent != "" {
		createPageOpts.Parent = &types.Parent{
			Type: opts.parentType,
			ID:   gotion.ExtractPageID(opts.parent),
		}
	}

	// Create client
	client, err := notion.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	// Create page
	result, err := client.CreatePage(ctx, createPageOpts)
	if err != nil {
		return fmt.Errorf("failed to create page: %w", err)
	}

	fmt.Println(string(result.RawJSON))
	return nil
}
