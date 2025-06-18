package packets

// body for registering
type SignupRequest struct {
	Email    string  `json:"email" binding:"required,email"`
	Password string  `json:"password" binding:"required,min=8"`
	Name     *string `json:"name"`
}

// body for logging in
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type UpdateCurrentProfileRequest struct {
	Email string  `json:"email" binding:"required,email"`
	Name  *string `json:"name"`
}
