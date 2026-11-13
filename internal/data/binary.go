package data

// Binary is an arbitrary binary item.
type Binary struct {
	ItemCore
	// Data is the binary payload.
	Data []byte
}

// NewBinary creates a new binary item with the given data and metadata.
//
// NewBinary stores a deep copy of the metadata.
func NewBinary(data []byte, meta Metadata) (Item, error) {
	core, err := newItem(TypeBinary)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, errEmptyData("data")
	}
	if err := meta.Validate(); err != nil {
		return nil, err
	}
	core.metadata = meta.Clone()
	return &Binary{
		ItemCore: core,
		Data:     data,
	}, nil
}
