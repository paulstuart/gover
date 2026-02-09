package model

// VersionData represents the data collected for a specific Go version.
type VersionData struct {
	Version   string           `json:"version"`
	ReleaseDate string           `json:"releaseDate,omitempty"` // Omit if empty for initial pass
	Changes   []ChangeCategory `json:"changes"`
}

// ChangeCategory represents a high-level category of changes (e.g., "Language Changes", "Core Library").
type ChangeCategory struct {
	Category    string        `json:"category"`
	Title       string        `json:"title,omitempty"`    // For specific language change titles
	Description string        `json:"description,omitempty"` // General description for the category or title
	Examples    []string      `json:"examples,omitempty"` // For extracted code examples
	Package     string        `json:"package,omitempty"`  // For "Core Library" changes
	Changes     []SymbolChange `json:"changes,omitempty"`  // For detailed symbol changes within a package
}

// SymbolChange represents a specific change to a function, method, or type within a package.
type SymbolChange struct {
	Type        string `json:"type"`        // e.g., "added", "changed", "obsoleted"
	Symbol      string `json:"symbol"`      // e.g., "http.NewRequestWithContext"
	Description string `json:"description"` // Description of the specific change
}
