package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/longkey1/gotion/internal/gotion"
	"github.com/longkey1/gotion/internal/gotion/config"
	"github.com/spf13/cobra"
)

type listOptions struct {
	query    string
	pageSize int
	format   string
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
	listCmd.Flags().StringVarP(&listOpts.format, "format", "f", "table", "Output format: json, text, table")
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

	client := gotion.NewClient(cfg.Token)

	// Validate and clamp page size
	pageSize := opts.pageSize
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	// Build search request
	searchReq := &gotion.SearchRequest{
		Query:    opts.query,
		PageSize: pageSize,
		Filter: &gotion.SearchFilter{
			Value:    "page",
			Property: "object",
		},
	}

	if opts.cursor != "" {
		searchReq.StartCursor = opts.cursor
	}

	// Set sort order
	if opts.sort == "ascending" || opts.sort == "descending" {
		searchReq.Sort = &gotion.SearchSort{
			Direction: opts.sort,
			Timestamp: "last_edited_time",
		}
	}

	resp, err := client.Search(ctx, searchReq)
	if err != nil {
		return fmt.Errorf("failed to search: %w", err)
	}

	formatter := gotion.NewFormatter(gotion.OutputFormat(opts.format), os.Stdout)
	return formatter.FormatPages(resp.Results, resp.NextCursor, resp.HasMore)
}
