package packets

// RESPONSES FOR /api/tv/screens/*

// screenResponse mirrors model.Screen but flattens times to RFC3339
type ScreenResponse struct {
	ID        int     `json:"id"`
	DeviceID  *string `json:"device_id"`
	Name      string  `json:"name"`
	Location  *string `json:"location"`
	Paired    bool    `json:"paired"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

type ContentResponse struct {
	ID        		int    	`json:"id"`
	Name      		string 	`json:"name"`
	Type      		string 	`json:"type"`
	URL       		string 	`json:"url"`
	DefaultDuration int 	`json:"default_duration"`
	UpdatedAt 		string 	`json:"updated_at"`
	CreatedAt 		string 	`json:"created_at"`
}
