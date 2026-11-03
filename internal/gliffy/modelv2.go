package gliffy

type GliffyDiagramV2 struct {
	ContentType    string         `json:"contentType"`
	Version        string         `json:"version"` // Note: string, not float64
	Title          string         `json:"title"`
	Revision       int            `json:"revision"`
	DefaultPage    string         `json:"defaultPage"`
	LastSerialized int64          `json:"lastSerialized"`
	Libraries      []string       `json:"libraries,omitempty"`
	EditorSettings map[string]any `json:"editorSettings,omitempty"`
	PrintModel     map[string]any `json:"printModel,omitempty"`
	Resources      []Resource     `json:"resources,omitempty"`
	Pages          []Page         `json:"pages"`
}

type Page struct {
	Type     string `json:"type"`
	ID       string `json:"id"`
	Title    string `json:"title"`
	Index    int    `json:"index"`
	External bool   `json:"external"`
	Scene    Scene  `json:"scene"`
}

type Scene struct {
	Width           float64        `json:"width"`
	Height          float64        `json:"height"`
	Background      string         `json:"background"`
	Objects         []*Object      `json:"objects"` // All shapes, lines, and nodes live here
	Layers          []Layer        `json:"layers"`  // Layer metadata definitions
	ImageCache      map[string]any `json:"imageCache,omitempty"`
	GridOn          bool           `json:"gridOn"`
	DrawingGuidesOn bool           `json:"drawingGuidesOn"`
}
