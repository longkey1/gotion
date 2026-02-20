package gotion

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// OutputFormat represents the output format type
type OutputFormat string

const (
	FormatJSON  OutputFormat = "json"
	FormatText  OutputFormat = "text"
	FormatTable OutputFormat = "table"
)

// Formatter handles output formatting
type Formatter struct {
	format OutputFormat
	writer io.Writer
}

// NewFormatter creates a new formatter
func NewFormatter(format OutputFormat, writer io.Writer) *Formatter {
	return &Formatter{
		format: format,
		writer: writer,
	}
}

// FormatPage formats a single page
func (f *Formatter) FormatPage(page *Page) error {
	switch f.format {
	case FormatJSON:
		return f.formatJSON(page)
	case FormatText:
		return f.formatPageText(page)
	case FormatTable:
		return f.formatPageTable(page)
	default:
		return f.formatPageText(page)
	}
}

// FormatPages formats multiple pages
func (f *Formatter) FormatPages(pages []Page, nextCursor string, hasMore bool) error {
	switch f.format {
	case FormatJSON:
		return f.formatJSON(map[string]interface{}{
			"results":     pages,
			"next_cursor": nextCursor,
			"has_more":    hasMore,
		})
	case FormatText:
		return f.formatPagesText(pages, nextCursor, hasMore)
	case FormatTable:
		return f.formatPagesTable(pages, nextCursor, hasMore)
	default:
		return f.formatPagesTable(pages, nextCursor, hasMore)
	}
}

// formatJSON outputs as JSON
func (f *Formatter) formatJSON(v interface{}) error {
	encoder := json.NewEncoder(f.writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(v)
}

// formatPageText formats a page as text
func (f *Formatter) formatPageText(page *Page) error {
	title := page.GetTitle()
	if title == "" {
		title = "(Untitled)"
	}

	fmt.Fprintf(f.writer, "Title: %s\n", title)
	fmt.Fprintf(f.writer, "ID: %s\n", page.ID)
	fmt.Fprintf(f.writer, "URL: %s\n", page.URL)
	fmt.Fprintf(f.writer, "Created: %s\n", page.CreatedTime.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(f.writer, "Last edited: %s\n", page.LastEditedTime.Format("2006-01-02 15:04:05"))

	if page.Icon != nil && page.Icon.Type == "emoji" {
		fmt.Fprintf(f.writer, "Icon: %s\n", page.Icon.Emoji)
	}

	fmt.Fprintln(f.writer, "\nProperties:")
	for name, prop := range page.Properties {
		if prop.Type == "title" {
			continue // Already shown as title
		}
		value := page.GetPropertyValue(name)
		if value != "" {
			fmt.Fprintf(f.writer, "  %s: %s\n", name, value)
		}
	}

	return nil
}

// formatPageTable formats a page as a table
func (f *Formatter) formatPageTable(page *Page) error {
	title := page.GetTitle()
	if title == "" {
		title = "(Untitled)"
	}

	// Simple key-value table
	rows := [][]string{
		{"Title", title},
		{"ID", page.ID},
		{"URL", page.URL},
		{"Created", page.CreatedTime.Format("2006-01-02 15:04:05")},
		{"Last edited", page.LastEditedTime.Format("2006-01-02 15:04:05")},
	}

	for name, prop := range page.Properties {
		if prop.Type == "title" {
			continue
		}
		value := page.GetPropertyValue(name)
		if value != "" {
			rows = append(rows, []string{name, value})
		}
	}

	return f.printTable([]string{"Property", "Value"}, rows)
}

// formatPagesText formats pages as text
func (f *Formatter) formatPagesText(pages []Page, nextCursor string, hasMore bool) error {
	for i, page := range pages {
		if i > 0 {
			fmt.Fprintln(f.writer, "---")
		}
		title := page.GetTitle()
		if title == "" {
			title = "(Untitled)"
		}
		fmt.Fprintf(f.writer, "%s\n", title)
		fmt.Fprintf(f.writer, "  ID: %s\n", page.ID)
		fmt.Fprintf(f.writer, "  URL: %s\n", page.URL)
		fmt.Fprintf(f.writer, "  Last edited: %s\n", page.LastEditedTime.Format("2006-01-02 15:04:05"))
	}

	if hasMore && nextCursor != "" {
		fmt.Fprintf(f.writer, "\n(More results available. Use --cursor %s to continue)\n", nextCursor)
	}

	return nil
}

// formatPagesTable formats pages as a table
func (f *Formatter) formatPagesTable(pages []Page, nextCursor string, hasMore bool) error {
	headers := []string{"Title", "ID", "Last Edited"}
	var rows [][]string

	for _, page := range pages {
		title := page.GetTitle()
		if title == "" {
			title = "(Untitled)"
		}
		// Truncate long titles
		if len(title) > 50 {
			title = title[:47] + "..."
		}
		rows = append(rows, []string{
			title,
			page.ID,
			page.LastEditedTime.Format("2006-01-02 15:04"),
		})
	}

	if err := f.printTable(headers, rows); err != nil {
		return err
	}

	if hasMore && nextCursor != "" {
		fmt.Fprintf(f.writer, "\n(More results available. Use --cursor %s to continue)\n", nextCursor)
	}

	return nil
}

// printTable prints a simple table
func (f *Formatter) printTable(headers []string, rows [][]string) error {
	if len(headers) == 0 {
		return nil
	}

	// Calculate column widths
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	// Print header
	f.printRow(headers, widths)
	f.printSeparator(widths)

	// Print rows
	for _, row := range rows {
		f.printRow(row, widths)
	}

	return nil
}

// printRow prints a table row
func (f *Formatter) printRow(cells []string, widths []int) {
	for i, cell := range cells {
		if i < len(widths) {
			fmt.Fprintf(f.writer, "%-*s", widths[i], cell)
			if i < len(cells)-1 {
				fmt.Fprint(f.writer, "  ")
			}
		}
	}
	fmt.Fprintln(f.writer)
}

// printSeparator prints a table separator
func (f *Formatter) printSeparator(widths []int) {
	for i, w := range widths {
		fmt.Fprint(f.writer, strings.Repeat("-", w))
		if i < len(widths)-1 {
			fmt.Fprint(f.writer, "  ")
		}
	}
	fmt.Fprintln(f.writer)
}
