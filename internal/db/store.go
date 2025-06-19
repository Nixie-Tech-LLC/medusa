package db

import (
	"github.com/jmoiron/sqlx"
	"github.com/Nixie-Tech-LLC/medusa/internal/model"
)

// Store defines all operations against the database.
type Store interface {
	// user functions
	CreateUser(email, hashedPassword string, name *string) (int, error)
	GetUserByEmail(email string) (*model.User, error)
	GetUserByID(id int) (*model.User, error)
	UpdateUserProfile(id int, email string, name *string) error

	// screen functions
	GetScreenByID(id int) (model.Screen, error)
	ListScreens() ([]model.Screen, error)
    CreateScreen(name string, location *string, createdBy int) (model.Screen, error)
	UpdateScreen(id int, name, location *string) error
	DeleteScreen(id int) error
	AssignScreenToUser(screenID, userID int) error
	AssignDeviceIDToScreen(screenID int, deviceID *string) error

	// content functions
	AssignContentToScreen(screenID, contentID int) error
	GetContentForScreen(screenID int) (*model.Content, error)
    CreateContent(name, typ, url string, metadata []byte, defaultDuration, createdBy int) (model.Content, error)

	GetContentByID(id int) (model.Content, error)
	ListContent() ([]model.Content, error)
	UpdateContent(id int, name, url *string, metadata *[]byte, defaultDuration *int) error
	DeleteContent(id int) error

	// playlists
    CreatePlaylist(name, description string, createdBy int) (model.Playlist, error)
	GetPlaylistByID(id int) (model.Playlist, error)
	ListPlaylists() ([]model.Playlist, error)
	UpdatePlaylist(id int, name, description *string) error
	DeletePlaylist(id int) error

	// playlist items
	AddItemToPlaylist(playlistID, contentID, position, duration int) (model.PlaylistItem, error)
	UpdatePlaylistItem(itemID int, position, duration *int) error
	RemovePlaylistItem(itemID int) error
	ListPlaylistItems(playlistID int) ([]model.PlaylistItem, error)

	// screen ↔ playlist
	AssignPlaylistToScreen(screenID, playlistID int) error
	GetPlaylistForScreen(screenID int) (model.Playlist, error)
}

// pgStore is the SQL-backed implementation of Store.
type pgStore struct {
	db *sqlx.DB
}

// compile-time check that *pgStore implements Store
var _ Store = (*pgStore)(nil)

// NewStore constructs a Store backed by the given sqlx.DB instance.
func NewStore(db *sqlx.DB) Store {
	return &pgStore{db: db}
}

// @ User
func (s *pgStore) CreateUser(email, hashedPassword string, name *string) (int, error) {
	return CreateUser(email, hashedPassword, name)
}
func (s *pgStore) GetUserByEmail(email string) (*model.User, error) {
	return GetUserByEmail(email)
}
func (s *pgStore) GetUserByID(id int) (*model.User, error) {
	return GetUserByID(id)
}
func (s *pgStore) UpdateUserProfile(id int, email string, name *string) error {
	return UpdateUserProfile(id, email, name)
}

// @ Screen
func (s *pgStore) GetScreenByID(id int) (model.Screen, error) {
	return GetScreenByID(id)
}
func (s *pgStore) ListScreens() ([]model.Screen, error) {
	return ListScreens()
}
func (s *pgStore) CreateScreen(name string, location *string, createdBy int) (model.Screen, error) {
    return CreateScreen(name, location, createdBy)
}
func (s *pgStore) UpdateScreen(id int, name, location *string) error {
	return UpdateScreen(id, name, location)
}
func (s *pgStore) DeleteScreen(id int) error {
	return DeleteScreen(id)
}
func (s *pgStore) AssignScreenToUser(screenID, userID int) error {
	return AssignScreenToUser(screenID, userID)
}
func (s *pgStore) AssignDeviceIDToScreen(screenID int, deviceID *string) error {
	return AssignDeviceIDToScreen(screenID, deviceID)
}

// @ Content
func (s *pgStore) AssignContentToScreen(screenID, contentID int) error {
	return AssignContentToScreen(screenID, contentID)
}
func (s *pgStore) GetContentForScreen(screenID int) (*model.Content, error) {
	return GetContentForScreen(screenID)
}
func (s *pgStore) CreateContent(
    name, typ, url string,
    metadata []byte,
    defaultDuration, createdBy int,
) (model.Content, error) {
    return CreateContent(name, typ, url, metadata, defaultDuration, createdBy)
}
func (s *pgStore) GetContentByID(id int) (model.Content, error) {
	return GetContentByID(id)
}
func (s *pgStore) ListContent() ([]model.Content, error) {
	return ListContent()
}
func (s *pgStore) UpdateContent(id int, name, url *string, metadata *[]byte, defaultDuration *int) error {
	return UpdateContent(id, name, url, metadata, defaultDuration)
}
func (s *pgStore) DeleteContent(id int) error {
	return DeleteContent(id)
}

// @ Playlist 
func (s *pgStore) CreatePlaylist(name, description string, createdBy int) (model.Playlist, error) {
    return CreatePlaylist(name, description, createdBy)
}
func (s *pgStore) GetPlaylistByID(id int) (model.Playlist, error) {
	return GetPlaylistByID(id)
}
func (s *pgStore) ListPlaylists() ([]model.Playlist, error) {
	return ListPlaylists()
}
func (s *pgStore) UpdatePlaylist(id int, name, description *string) error {
	return UpdatePlaylist(id, name, description)
}
func (s *pgStore) DeletePlaylist(id int) error {
	return DeletePlaylist(id)
}

// @ Playlist Item
func (s *pgStore) AddItemToPlaylist(playlistID, contentID, position, duration int) (model.PlaylistItem, error) {
	return AddItemToPlaylist(playlistID, contentID, position, duration)
}
func (s *pgStore) UpdatePlaylistItem(itemID int, position, duration *int) error {
	return UpdatePlaylistItem(itemID, position, duration)
}
func (s *pgStore) RemovePlaylistItem(itemID int) error {
	return RemovePlaylistItem(itemID)
}
func (s *pgStore) ListPlaylistItems(playlistID int) ([]model.PlaylistItem, error) {
	return ListPlaylistItems(playlistID)
}

// @ Screen <-> Playlist
func (s *pgStore) AssignPlaylistToScreen(screenID, playlistID int) error {
	return AssignPlaylistToScreen(screenID, playlistID)
}
func (s *pgStore) GetPlaylistForScreen(screenID int) (model.Playlist, error) {
	return GetPlaylistForScreen(screenID)
}

