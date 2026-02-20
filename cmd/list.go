package cmd

import (
	"context"
	"fmt"

	"github.com/longkey1/gotion/internal/gotion/config"
	"github.com/longkey1/gotion/internal/notion"
	"github.com/spf13/cobra"
)

type listOptions struct {
	query    string
	pageSize int
	sort     string
	cursor   string
}

var listOpts = &listOptions{}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "Search and list Notion pages",
	Long:  `Search for pages in Notion and display the results.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runList(cmd.Context(), listOpts)
	},
}

func init() {
	listCmd.Flags().StringVarP(&listOpts.query, "query", "q", "", "Search keyword")
	listCmd.Flags().IntVarP(&listOpts.pageSize, "page-size", "n", 10, "Number of results to retrieve (max 100)")
	listCmd.Flags().StringVar(&listOpts.sort, "sort", "descending", "Sort order: ascending, descending")
	listCmd.Flags().StringVar(&listOpts.cursor, "cursor", "", "Pagination cursor")

	rootCmd.AddCommand(listCmd)
}

func runList(ctx context.Context, opts *listOptions) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return err
	}

	// Create client based on backend
	client, err := notion.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	// Validate and clamp page size
	pageSize := opts.pageSize
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	// Build search options
	searchOpts := &notion.SearchOptions{
		PageSize:    pageSize,
		StartCursor: opts.cursor,
		Sort:        opts.sort,
	}

	result, err := client.Search(ctx, opts.query, searchOpts)
	if err != nil {
		return fmt.Errorf("failed to search: %w", err)
	}

	// Format output
	output, err := client.FormatSearch(result)
	if err != nil {
		return err
	}

	fmt.Print(output)
	return nil
}
