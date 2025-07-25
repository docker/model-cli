package desktop

// ProgressMessage represents a structured message for progress reporting
type ProgressMessage struct {
	Type    string `json:"type"`    // "progress", "success", or "error"
	Message string `json:"message"` // Deprecated: the message should be defined by clients based on Message.Total and Message.Layer
	Total   uint64 `json:"total"`
	Pulled  uint64 `json:"pulled"` // Deprecated: use Layer.Current
	Layer   Layer  `json:"layer"`  // Current layer information
}

type Layer struct {
	ID      string // Layer ID
	Size    uint64 // Layer size
	Current uint64 // Current bytes transferred
}

type OpenAIChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenAIChatRequest struct {
	Model    string              `json:"model"`
	Messages []OpenAIChatMessage `json:"messages"`
	Stream   bool                `json:"stream"`
}

type OpenAIChatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
			Role    string `json:"role,omitempty"`
		} `json:"delta"`
		Index        int    `json:"index"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}
