package db

import (
	"github.com/Nixie-Tech-LLC/medusa/internal/model"
	"github.com/jmoiron/sqlx"
	"time"
)

// ContentItem represents a content item with URL and duration
type ContentItem struct {
	URL      string `db:"url"`
	Duration int    `db:"duration"`
	Type     string `db:"type"`
}

// Store defines all operations against the database.
type Store interface {
	// user functions
	CreateUser(email, hashedPassword string, name *string) (int, error)
	GetUserByEmail(email string) (*model.User, error)
	GetUserByID(id int) (*model.User, error)
	UpdateUserProfile(id int, email string, name *string) error

	// screen functions
	CreateScreen(name string, location *string, createdBy int) (model.Screen, error)
	UpdateScreen(id int, name, location *string) error
	GetScreenByID(id int) (model.Screen, error)
	DeleteScreen(id int) error

	ListScreens() ([]model.Screen, error)
	AssignScreenToUser(screenID, userID int) error
	AssignDeviceIDToScreen(screenID int, deviceID *string) error
	UpdateClientInformation(screenID int, clientInformation *string) error
	UpdateClientDimensions(screenID int, width, height int) error

	IsScreenPairedByDeviceID(deviceID *string) (bool, error)
	GetScreenByDeviceID(deviceID *string) (model.Screen, error)

	// content functions
	CreateContent(name, typ, url string, resWidth int, resHeight int, createdBy int) (model.Content, error)
	GetContentByID(id int) (model.Content, error)
	UpdateContent(id int, name, url *string, width int, height int) error
	DeleteContent(id int) error

	ListContent() ([]model.Content, error)
	SearchContent(name, contentType *string, createdBy *int) ([]model.Content, error)
	SearchContentMultiple(names, types []string, createdBy *int) ([]model.Content, error)

	// playlists
	CreatePlaylist(name, description string, createdBy int) (model.Playlist, error)
	GetPlaylistByID(id int) (model.Playlist, error)
	UpdatePlaylist(id int, name, description *string) error
	DeletePlaylist(id int) error

	ListPlaylists() ([]model.Playlist, error)

	// playlist items
	AddItemToPlaylist(playlistID, contentID, position, duration int) (model.PlaylistItem, error)
	UpdatePlaylistItem(itemID int, position, duration *int) error
	RemovePlaylistItem(itemID int) error

	ListPlaylistItems(playlistID int) ([]model.PlaylistItem, error)
	ReorderPlaylistItems(playlistID int, itemIDs []int) error

	// screen ↔ playlist
	AssignPlaylistToScreen(screenID, playlistID int) error
	GetPlaylistForScreen(screenID int) (model.Playlist, error)
	GetScreensUsingPlaylist(playlistID int) ([]model.Screen, error)
	GetPlaylistContentForScreen(screenID int) (string, []ContentItem, error)

	CreateSchedule(name string, createdBy int) (model.Schedule, error)
	DeleteSchedule(scheduleID int) error
	GetScheduleByID(scheduleID int) (model.Schedule, error)

	AssignScheduleToScreen(scheduleID, screenID int) error
	UnassignScheduleFromScreen(scheduleID, screenID int) error
	ListSchedules(ownerID int) ([]model.Schedule, error)

	CreateScheduleWindow(scheduleID, playlistID int, start, end time.Time, recurrence string, recurUntil *time.Time, priority int) (model.ScheduleWindow, error)
	DeleteScheduleWindowAll(windowID int) error
	DeleteScheduleWindowOneOccurrence(windowID int, occurStart time.Time) error
	ListScheduleOccurrences(scheduleID int, from, to time.Time) ([]model.ScheduleOccurrence, error)
	GetScheduleByWindowID(windowID int) (model.Schedule, error)

    ResolvePlaylistForScreenAt(screenID int, at time.Time) (int, error)
	GetEffectivePlaylistForScreen(screenID int, now time.Time) (model.Playlist, []ContentItem, string, error)
	GetPlaylistContentByPlaylistID(playlistID int) (string, []ContentItem, error)
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
func (s *pgStore) UpdateClientInformation(screenID int, clientInformation *string) error {
	return UpdateClientInformation(screenID, clientInformation)
}
func (s *pgStore) UpdateClientDimensions(screenID int, width, height int) error {
	return UpdateClientDimensions(screenID, width, height)
}
func (s *pgStore) IsScreenPairedByDeviceID(deviceID *string) (bool, error) {
	return IsScreenPairedByDeviceID(deviceID)
}

// @ Content
func (s *pgStore) CreateContent(
	name, typ, url string,
	resWidth, resHeight, createdBy int,
) (model.Content, error) {
	return CreateContent(name, typ, url, resWidth, resHeight, createdBy)
}
func (s *pgStore) GetContentByID(id int) (model.Content, error) {
	return GetContentByID(id)
}
func (s *pgStore) ListContent() ([]model.Content, error) {
	return ListContent()
}
func (s *pgStore) SearchContent(name, contentType *string, createdBy *int) ([]model.Content, error) {
	return SearchContent(name, contentType, createdBy)
}
func (s *pgStore) SearchContentMultiple(names, types []string, createdBy *int) ([]model.Content, error) {
	return SearchContentMultiple(names, types, createdBy)
}
func (s *pgStore) UpdateContent(id int, name, url *string, width int, height int) error {
	return UpdateContent(id, name, url, width, height)
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
func (s *pgStore) ReorderPlaylistItems(playlistID int, itemIDs []int) error {
	return ReorderPlaylistItems(playlistID, itemIDs)
}

// @ Screen <-> Playlist
func (s *pgStore) AssignPlaylistToScreen(screenID, playlistID int) error {
	return AssignPlaylistToScreen(screenID, playlistID)
}
func (s *pgStore) GetPlaylistForScreen(screenID int) (model.Playlist, error) {
	return GetPlaylistForScreen(screenID)
}
func (s *pgStore) GetScreensUsingPlaylist(playlistID int) ([]model.Screen, error) {
	return GetScreensUsingPlaylist(playlistID)
}
func (s *pgStore) GetPlaylistContentForScreen(screenID int) (string, []ContentItem, error) {
	return GetPlaylistContentForScreen(screenID)
}

// @ Schedules
func (s *pgStore) CreateSchedule(name string, createdBy int) (model.Schedule, error) {
	return CreateSchedule(name, createdBy)
}
func (s *pgStore) DeleteSchedule(scheduleID int) error { return DeleteSchedule(scheduleID) }
func (s *pgStore) ListSchedules(ownerID int) ([]model.Schedule, error) { return ListSchedules(ownerID) }
func (s *pgStore) GetScheduleByID(scheduleID int) (model.Schedule, error) { return GetSchedule(scheduleID) }

func (s *pgStore) AssignScheduleToScreen(scheduleID, screenID int) error {
	return AssignScheduleToScreen(scheduleID, screenID)
}
func (s *pgStore) UnassignScheduleFromScreen(scheduleID, screenID int) error {
	return UnassignScheduleFromScreen(scheduleID, screenID)
}

func (s *pgStore) CreateScheduleWindow(scheduleID, playlistID int, start, end time.Time, recurrence string, recurUntil *time.Time, priority int) (model.ScheduleWindow, error) {
	return CreateScheduleWindow(scheduleID, playlistID, start, end, recurrence, recurUntil, priority)
}
func (s *pgStore) DeleteScheduleWindowAll(windowID int) error {
	return DeleteScheduleWindowAll(windowID)
}
func (s *pgStore) DeleteScheduleWindowOneOccurrence(windowID int, occurStart time.Time) error {
	return DeleteScheduleWindowOneOccurrence(windowID, occurStart)
}
func (s *pgStore) ListScheduleOccurrences(scheduleID int, from, to time.Time) ([]model.ScheduleOccurrence, error) {
	return ListScheduleOccurrences(scheduleID, from, to)
}
func (s *pgStore) GetScheduleByWindowID(windowID int) (model.Schedule, error) {
	return GetScheduleByWindowID(windowID)
}

func (s *pgStore) ResolvePlaylistForScreenAt(screenID int, at time.Time) (int, error) {
	return ResolvePlaylistForScreenAt(screenID, at)
}
func (s *pgStore) GetScreenByDeviceID(deviceID *string) (model.Screen, error) {
	return GetScreenByDeviceID(deviceID)
}
func (s *pgStore) GetEffectivePlaylistForScreen(screenID int, now time.Time) (model.Playlist, []ContentItem, string, error) {
	return GetEffectivePlaylistForScreen(screenID, now)
}
func (s *pgStore) GetPlaylistContentByPlaylistID(playlistID int) (string, []ContentItem, error) {
	return GetPlaylistContentByPlaylistID(playlistID)
}


