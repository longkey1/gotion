package gotion

import (
	"time"
)

// SearchResponse represents the response from the search API
type SearchResponse struct {
	Object     string   `json:"object"`
	Results    []Page   `json:"results"`
	NextCursor string   `json:"next_cursor"`
	HasMore    bool     `json:"has_more"`
}

// Page represents a Notion page
type Page struct {
	Object         string                 `json:"object"`
	ID             string                 `json:"id"`
	CreatedTime    time.Time              `json:"created_time"`
	LastEditedTime time.Time              `json:"last_edited_time"`
	CreatedBy      User                   `json:"created_by"`
	LastEditedBy   User                   `json:"last_edited_by"`
	Cover          *File                  `json:"cover"`
	Icon           *Icon                  `json:"icon"`
	Parent         Parent                 `json:"parent"`
	Archived       bool                   `json:"archived"`
	InTrash        bool                   `json:"in_trash"`
	Properties     map[string]Property    `json:"properties"`
	URL            string                 `json:"url"`
	PublicURL      *string                `json:"public_url"`
}

// User represents a Notion user
type User struct {
	Object    string  `json:"object"`
	ID        string  `json:"id"`
	Name      string  `json:"name,omitempty"`
	AvatarURL *string `json:"avatar_url,omitempty"`
	Type      string  `json:"type,omitempty"`
	Person    *Person `json:"person,omitempty"`
	Bot       *Bot    `json:"bot,omitempty"`
}

// Person represents a person user
type Person struct {
	Email string `json:"email"`
}

// Bot represents a bot user
type Bot struct {
	Owner         Owner  `json:"owner"`
	WorkspaceName string `json:"workspace_name"`
}

// Owner represents the owner of a bot
type Owner struct {
	Type      string `json:"type"`
	Workspace bool   `json:"workspace"`
}

// File represents a file object
type File struct {
	Type     string    `json:"type"`
	External *External `json:"external,omitempty"`
	File     *FileData `json:"file,omitempty"`
}

// External represents an external file
type External struct {
	URL string `json:"url"`
}

// FileData represents file data
type FileData struct {
	URL        string    `json:"url"`
	ExpiryTime time.Time `json:"expiry_time"`
}

// Icon represents an icon (emoji or file)
type Icon struct {
	Type     string    `json:"type"`
	Emoji    string    `json:"emoji,omitempty"`
	External *External `json:"external,omitempty"`
	File     *FileData `json:"file,omitempty"`
}

// Parent represents the parent of a page
type Parent struct {
	Type       string `json:"type"`
	DatabaseID string `json:"database_id,omitempty"`
	PageID     string `json:"page_id,omitempty"`
	Workspace  bool   `json:"workspace,omitempty"`
	BlockID    string `json:"block_id,omitempty"`
}

// Property represents a page property
type Property struct {
	ID          string       `json:"id"`
	Type        string       `json:"type"`
	Title       []RichText   `json:"title,omitempty"`
	RichText    []RichText   `json:"rich_text,omitempty"`
	Number      *float64     `json:"number,omitempty"`
	Select      *SelectValue `json:"select,omitempty"`
	MultiSelect []SelectValue `json:"multi_select,omitempty"`
	Date        *DateValue   `json:"date,omitempty"`
	People      []User       `json:"people,omitempty"`
	Files       []File       `json:"files,omitempty"`
	Checkbox    *bool        `json:"checkbox,omitempty"`
	URL         *string      `json:"url,omitempty"`
	Email       *string      `json:"email,omitempty"`
	PhoneNumber *string      `json:"phone_number,omitempty"`
	Formula     *Formula     `json:"formula,omitempty"`
	Relation    []Relation   `json:"relation,omitempty"`
	Rollup      *Rollup      `json:"rollup,omitempty"`
	CreatedTime *time.Time   `json:"created_time,omitempty"`
	CreatedBy   *User        `json:"created_by,omitempty"`
	LastEditedTime *time.Time `json:"last_edited_time,omitempty"`
	LastEditedBy   *User      `json:"last_edited_by,omitempty"`
	Status      *StatusValue `json:"status,omitempty"`
	UniqueID    *UniqueID    `json:"unique_id,omitempty"`
}

// RichText represents rich text content
type RichText struct {
	Type        string       `json:"type"`
	Text        *TextContent `json:"text,omitempty"`
	Mention     *Mention     `json:"mention,omitempty"`
	Equation    *Equation    `json:"equation,omitempty"`
	Annotations *Annotations `json:"annotations,omitempty"`
	PlainText   string       `json:"plain_text"`
	Href        *string      `json:"href,omitempty"`
}

// TextContent represents text content
type TextContent struct {
	Content string `json:"content"`
	Link    *Link  `json:"link,omitempty"`
}

// Link represents a link
type Link struct {
	URL string `json:"url"`
}

// Mention represents a mention
type Mention struct {
	Type            string           `json:"type"`
	User            *User            `json:"user,omitempty"`
	Page            *PageReference   `json:"page,omitempty"`
	Database        *DatabaseRef     `json:"database,omitempty"`
	Date            *DateValue       `json:"date,omitempty"`
	LinkPreview     *LinkPreview     `json:"link_preview,omitempty"`
	TemplateMention *TemplateMention `json:"template_mention,omitempty"`
}

// PageReference represents a page reference
type PageReference struct {
	ID string `json:"id"`
}

// DatabaseRef represents a database reference
type DatabaseRef struct {
	ID string `json:"id"`
}

// LinkPreview represents a link preview
type LinkPreview struct {
	URL string `json:"url"`
}

// TemplateMention represents a template mention
type TemplateMention struct {
	Type             string `json:"type"`
	TemplateMentionDate string `json:"template_mention_date,omitempty"`
	TemplateMentionUser string `json:"template_mention_user,omitempty"`
}

// Equation represents an equation
type Equation struct {
	Expression string `json:"expression"`
}

// Annotations represents text annotations
type Annotations struct {
	Bold          bool   `json:"bold"`
	Italic        bool   `json:"italic"`
	Strikethrough bool   `json:"strikethrough"`
	Underline     bool   `json:"underline"`
	Code          bool   `json:"code"`
	Color         string `json:"color"`
}

// SelectValue represents a select value
type SelectValue struct {
	ID    string `json:"id,omitempty"`
	Name  string `json:"name"`
	Color string `json:"color,omitempty"`
}

// StatusValue represents a status value
type StatusValue struct {
	ID    string `json:"id,omitempty"`
	Name  string `json:"name"`
	Color string `json:"color,omitempty"`
}

// DateValue represents a date value
type DateValue struct {
	Start    string  `json:"start"`
	End      *string `json:"end,omitempty"`
	TimeZone *string `json:"time_zone,omitempty"`
}

// Formula represents a formula result
type Formula struct {
	Type    string   `json:"type"`
	String  *string  `json:"string,omitempty"`
	Number  *float64 `json:"number,omitempty"`
	Boolean *bool    `json:"boolean,omitempty"`
	Date    *DateValue `json:"date,omitempty"`
}

// Relation represents a relation
type Relation struct {
	ID string `json:"id"`
}

// Rollup represents a rollup result
type Rollup struct {
	Type   string      `json:"type"`
	Number *float64    `json:"number,omitempty"`
	Date   *DateValue  `json:"date,omitempty"`
	Array  []Property  `json:"array,omitempty"`
}

// UniqueID represents a unique ID
type UniqueID struct {
	Prefix *string `json:"prefix,omitempty"`
	Number int     `json:"number"`
}

// APIError represents an error from the Notion API
type APIError struct {
	Object  string `json:"object"`
	Status  int    `json:"status"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *APIError) Error() string {
	return e.Message
}

// SearchRequest represents a search request
type SearchRequest struct {
	Query       string       `json:"query,omitempty"`
	Sort        *SearchSort  `json:"sort,omitempty"`
	Filter      *SearchFilter `json:"filter,omitempty"`
	StartCursor string       `json:"start_cursor,omitempty"`
	PageSize    int          `json:"page_size,omitempty"`
}

// SearchSort represents sort options for search
type SearchSort struct {
	Direction string `json:"direction"`
	Timestamp string `json:"timestamp"`
}

// SearchFilter represents filter options for search
type SearchFilter struct {
	Value    string `json:"value"`
	Property string `json:"property"`
}
