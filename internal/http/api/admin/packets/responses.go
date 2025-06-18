package packets

// Response mirrors model.Content but flattens time.
type ContentResponse struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	URL       string `json:"url"`
	CreatedAt string `json:"created_at"`
}
