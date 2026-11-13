package data

import "strings"

// Text is an arbitrary text item.
type Text struct {
	ItemCore
	// Content is the text data.
	Content string
}

// NewText creates a new text item with the given content and metadata.
//
// NewText stores a deep copy of the metadata.
func NewText(content string, meta Metadata) (Item, error) {
	core, err := newItem(TypeText)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(content) == "" {
		return nil, errEmptyData("content")
	}
	if err := meta.Validate(); err != nil {
		return nil, err
	}
	core.metadata = meta.Clone()
	return &Text{
		ItemCore: core,
		Content:  content,
	}, nil
}
