package gotion

import (
	"strconv"
	"strings"
)

// GetTitle extracts the title from a page's properties
func (p *Page) GetTitle() string {
	for _, prop := range p.Properties {
		if prop.Type == "title" && len(prop.Title) > 0 {
			var sb strings.Builder
			for _, text := range prop.Title {
				sb.WriteString(text.PlainText)
			}
			return sb.String()
		}
	}
	return ""
}

// GetPropertyValue extracts a property value as a string
func (p *Page) GetPropertyValue(name string) string {
	prop, ok := p.Properties[name]
	if !ok {
		return ""
	}

	switch prop.Type {
	case "title":
		return extractPlainText(prop.Title)
	case "rich_text":
		return extractPlainText(prop.RichText)
	case "number":
		if prop.Number != nil {
			return formatNumber(*prop.Number)
		}
	case "select":
		if prop.Select != nil {
			return prop.Select.Name
		}
	case "multi_select":
		var names []string
		for _, s := range prop.MultiSelect {
			names = append(names, s.Name)
		}
		return strings.Join(names, ", ")
	case "date":
		if prop.Date != nil {
			if prop.Date.End != nil {
				return prop.Date.Start + " → " + *prop.Date.End
			}
			return prop.Date.Start
		}
	case "people":
		var names []string
		for _, u := range prop.People {
			names = append(names, u.Name)
		}
		return strings.Join(names, ", ")
	case "checkbox":
		if prop.Checkbox != nil {
			if *prop.Checkbox {
				return "✓"
			}
			return "✗"
		}
	case "url":
		if prop.URL != nil {
			return *prop.URL
		}
	case "email":
		if prop.Email != nil {
			return *prop.Email
		}
	case "phone_number":
		if prop.PhoneNumber != nil {
			return *prop.PhoneNumber
		}
	case "status":
		if prop.Status != nil {
			return prop.Status.Name
		}
	case "created_time":
		if prop.CreatedTime != nil {
			return prop.CreatedTime.Format("2006-01-02 15:04:05")
		}
	case "last_edited_time":
		if prop.LastEditedTime != nil {
			return prop.LastEditedTime.Format("2006-01-02 15:04:05")
		}
	case "created_by":
		if prop.CreatedBy != nil {
			return prop.CreatedBy.Name
		}
	case "last_edited_by":
		if prop.LastEditedBy != nil {
			return prop.LastEditedBy.Name
		}
	case "unique_id":
		if prop.UniqueID != nil {
			if prop.UniqueID.Prefix != nil {
				return *prop.UniqueID.Prefix + "-" + strconv.Itoa(prop.UniqueID.Number)
			}
			return strconv.Itoa(prop.UniqueID.Number)
		}
	}

	return ""
}

// extractPlainText extracts plain text from rich text array
func extractPlainText(texts []RichText) string {
	var sb strings.Builder
	for _, text := range texts {
		sb.WriteString(text.PlainText)
	}
	return sb.String()
}

// formatNumber formats a float64 as a string, removing trailing zeros
func formatNumber(n float64) string {
	s := strconv.FormatFloat(n, 'f', -1, 64)
	return s
}
