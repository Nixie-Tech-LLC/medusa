// exposes a Store interface that is passed to API calls w/ param requirements
package db

import (
	"github.com/Nixie-Tech-LLC/medusa/internal/model"
	"github.com/jmoiron/sqlx"
)

type Store interface {
	// user functions
	CreateUser(email, hashedPassword string, name *string) (int, error)
	GetUserByEmail(email string) (*model.User, error)
	GetUserByID(id int) (*model.User, error)
	UpdateUserProfile(id int, email string, name *string) error

	// screen functions
	GetScreenByID(id int) (model.Screen, error)
	ListScreens() ([]model.Screen, error)
	CreateScreen(name string, location *string) (model.Screen, error)
	UpdateScreen(id int, name, location *string) error
	DeleteScreen(id int) error
	AssignScreenToUser(screenID, userID int) error
	AssignDeviceIDToScreen(screenID int, deviceID *string) error

	// content functions
	CreateContent(name, ctype, url string) (model.Content, error)
	GetContentByID(id int) (*model.Content, error)
	ListContent() ([]model.Content, error)
	AssignContentToScreen(screenID, contentID int) error
	GetContentForScreen(screenID int) (*model.Content, error)
}

type pgStore struct {
	db *sqlx.DB
}

// compile-time check that pgStore implements Store
// required so linter doesn't complain
var _ Store = (*pgStore)(nil)

func NewStore() Store {
	return &pgStore{db: DB}
}

// @ USER

// shell function that points to ./db.go:CreateUser
func (s *pgStore) CreateUser(email, hashedPassword string, name *string) (int, error) {
	return CreateUser(email, hashedPassword, name)
}

// shell function that points to ./db.go:GetUserByEmail
func (s *pgStore) GetUserByEmail(email string) (*model.User, error) {
	return GetUserByEmail(email)
}

// shell function that points to ./db.go:GetUserByID
func (s *pgStore) GetUserByID(id int) (*model.User, error) {
	return GetUserByID(id)
}

// shell function that points to ./db.go:UpdateUserProfile
func (s *pgStore) UpdateUserProfile(id int, email string, name *string) error {
	return UpdateUserProfile(id, email, name)
}

// @ SCREEN

// shell function that points to ./db.go:GetScreenByID
func (s *pgStore) GetScreenByID(id int) (model.Screen, error) {
	return GetScreenByID(id)
}

// shell function that points to ./db.go:ListScreens
func (s *pgStore) ListScreens() ([]model.Screen, error) {
	return ListScreens()
}

// shell function that points to ./db.go:CreateScreen
func (s *pgStore) CreateScreen(name string, location *string) (model.Screen, error) {
	return CreateScreen(name, location)
}

// shell function that points to ./db.go:UpdateScreen
func (s *pgStore) UpdateScreen(id int, name, location *string) error {
	return UpdateScreen(id, name, location)
}

// shell function that points to ./db.go:DeleteScreen
func (s *pgStore) DeleteScreen(id int) error {
	return DeleteScreen(id)
}

// shell function that points to ./db.go:AssignScreenToUser
func (s *pgStore) AssignScreenToUser(screenID, userID int) error {
	return AssignScreenToUser(screenID, userID)
}

func (s *pgStore) AssignDeviceIDToScreen(screenID int, deviceID *string) error {
	return AssignDeviceIDToScreen(screenID, deviceID)
}

// @ CONTENT

func (s *pgStore) CreateContent(name, ctype, url string) (model.Content, error) {
	return CreateContent(name, ctype, url)
}

func (s *pgStore) GetContentByID(id int) (*model.Content, error) {
	return GetContentByID(id)
}

func (s *pgStore) ListContent() ([]model.Content, error) {
	return ListContent()
}

func (s *pgStore) AssignContentToScreen(screenID, contentID int) error {
	return AssignContentToScreen(screenID, contentID)
}

func (s *pgStore) GetContentForScreen(screenID int) (*model.Content, error) {
	return GetContentForScreen(screenID)
}
