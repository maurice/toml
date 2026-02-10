package toml

// Document represents a TOML document.
type Document struct {
	// implementation
}

// Parse reads a TOML document from bytes.
func Parse(_ []byte) (*Document, error) {
	return &Document{}, nil
}
