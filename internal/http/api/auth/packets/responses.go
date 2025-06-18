package packets

// returned for profile endpoints
type ProfileResponse struct {
	ID        int     `json:"id"`
	Email     string  `json:"email"`
	Name      *string `json:"name"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}
