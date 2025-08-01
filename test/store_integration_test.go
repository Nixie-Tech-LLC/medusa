package test

import (
	"context"
	"testing"

	"github.com/Nixie-Tech-LLC/medusa/internal/db"
	"github.com/Nixie-Tech-LLC/medusa/internal/redis"
	"github.com/stretchr/testify/assert"
)

// TestStoreIntegration tests the core business logic functions directly
func TestStoreIntegration(t *testing.T) {
	// Setup Redis
	redis.InitRedis("localhost:6379", "", "")

	store := db.NewStore(nil)

	t.Run("User Management", func(t *testing.T) {
		// Test user creation
		userID, err := store.CreateUser("test@example.com", "hashedpassword", nil)
		assert.NoError(t, err)
		assert.Greater(t, userID, 0)

		// Test user retrieval
		user, err := store.GetUserByEmail("test@example.com")
		assert.NoError(t, err)
		assert.Equal(t, "test@example.com", user.Email)

		// Test user update
		name := "Updated Name"
		err = store.UpdateUserProfile(userID, "newemail@example.com", &name)
		assert.NoError(t, err)
	})

	t.Run("Content Management", func(t *testing.T) {
		// Create test user first
		userID, _ := store.CreateUser("content@example.com", "password", nil)

		// Test content creation
		content, err := store.CreateContent("Test Video", "video", "https://example.com/video.mp4", 30, userID)
		assert.NoError(t, err)
		assert.Equal(t, "Test Video", content.Name)
		assert.Equal(t, "video", content.Type)

		// Test content retrieval
		retrievedContent, err := store.GetContentByID(content.ID)
		assert.NoError(t, err)
		assert.Equal(t, content.Name, retrievedContent.Name)

		// Test content listing
		contents, err := store.ListContent()
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(contents), 1)

		// Test content update
		newName := "Updated Video"
		newURL := "https://example.com/updated.mp4"
		newDuration := 60
		err = store.UpdateContent(content.ID, &newName, &newURL, &newDuration)
		assert.NoError(t, err)

		// Verify update
		updatedContent, _ := store.GetContentByID(content.ID)
		assert.Equal(t, newName, updatedContent.Name)
		assert.Equal(t, newURL, updatedContent.URL)
		assert.Equal(t, newDuration, updatedContent.DefaultDuration)
	})

	t.Run("Screen Management", func(t *testing.T) {
		// Create test user
		userID, _ := store.CreateUser("screen@example.com", "password", nil)

		// Test screen creation
		location := "Lobby"
		screen, err := store.CreateScreen("Main Display", &location, userID)
		assert.NoError(t, err)
		assert.Equal(t, "Main Display", screen.Name)
		assert.Equal(t, location, *screen.Location)

		// Test screen retrieval
		retrievedScreen, err := store.GetScreenByID(screen.ID)
		assert.NoError(t, err)
		assert.Equal(t, screen.Name, retrievedScreen.Name)

		// Test screen listing
		screens, err := store.ListScreens()
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(screens), 1)

		// Test screen update
		newName := "Updated Display"
		newLocation := "Conference Room"
		err = store.UpdateScreen(screen.ID, &newName, &newLocation)
		assert.NoError(t, err)

		// Verify update
		updatedScreen, _ := store.GetScreenByID(screen.ID)
		assert.Equal(t, newName, updatedScreen.Name)
		assert.Equal(t, newLocation, *updatedScreen.Location)
	})

	t.Run("Playlist Management", func(t *testing.T) {
		// Create test user
		userID, _ := store.CreateUser("playlist@example.com", "password", nil)

		// Test playlist creation
		description := "Test playlist for integration testing"
		playlist, err := store.CreatePlaylist("Test Playlist", description, userID)
		assert.NoError(t, err)
		assert.Equal(t, "Test Playlist", playlist.Name)
		assert.Equal(t, description, playlist.Description)

		// Test playlist retrieval
		retrievedPlaylist, err := store.GetPlaylistByID(playlist.ID)
		assert.NoError(t, err)
		assert.Equal(t, playlist.Name, retrievedPlaylist.Name)

		// Test playlist listing
		playlists, err := store.ListPlaylists()
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(playlists), 1)

		// Test playlist update
		newName := "Updated Playlist"
		newDescription := "Updated description"
		err = store.UpdatePlaylist(playlist.ID, &newName, &newDescription)
		assert.NoError(t, err)

		// Verify update
		updatedPlaylist, _ := store.GetPlaylistByID(playlist.ID)
		assert.Equal(t, newName, updatedPlaylist.Name)
		assert.Equal(t, newDescription, updatedPlaylist.Description)
	})

	t.Run("Playlist Items Management", func(t *testing.T) {
		// Create test user and content
		userID, _ := store.CreateUser("items@example.com", "password", nil)
		content, _ := store.CreateContent("Test Content", "image", "https://example.com/image.jpg", 15, userID)
		playlist, _ := store.CreatePlaylist("Test Playlist", "For testing items", userID)

		// Test adding item to playlist
		item, err := store.AddItemToPlaylist(playlist.ID, content.ID, 1, 20)
		assert.NoError(t, err)
		assert.Equal(t, playlist.ID, item.PlaylistID)
		assert.Equal(t, content.ID, item.ContentID)
		assert.Equal(t, 1, item.Position)
		assert.Equal(t, 20, item.Duration)

		// Test listing playlist items
		items, err := store.ListPlaylistItems(playlist.ID)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(items))
		assert.Equal(t, content.ID, items[0].ContentID)

		// Test updating playlist item
		newPosition := 2
		newDuration := 30
		err = store.UpdatePlaylistItem(item.ID, &newPosition, &newDuration)
		assert.NoError(t, err)

		// Verify update
		updatedItems, _ := store.ListPlaylistItems(playlist.ID)
		assert.Equal(t, newPosition, updatedItems[0].Position)
		assert.Equal(t, newDuration, updatedItems[0].Duration)

		// Test reordering playlist items
		content2, _ := store.CreateContent("Test Content 2", "video", "https://example.com/video2.mp4", 25, userID)
		item2, _ := store.AddItemToPlaylist(playlist.ID, content2.ID, 3, 25)

		// Reorder: put item2 first, then item
		err = store.ReorderPlaylistItems(playlist.ID, []int{item2.ID, item.ID})
		assert.NoError(t, err)

		// Verify reorder
		reorderedItems, _ := store.ListPlaylistItems(playlist.ID)
		assert.Equal(t, 2, len(reorderedItems))
		assert.Equal(t, item2.ID, reorderedItems[0].ID)
		assert.Equal(t, item.ID, reorderedItems[1].ID)
		assert.Equal(t, 1, reorderedItems[0].Position)
		assert.Equal(t, 2, reorderedItems[1].Position)
	})

	t.Run("Screen-Content Assignment", func(t *testing.T) {
		// Create test user, screen, and content
		userID, _ := store.CreateUser("assignment@example.com", "password", nil)
		screen, _ := store.CreateScreen("Test Screen", nil, userID)
		content, _ := store.CreateContent("Test Content", "image", "https://example.com/test.jpg", 10, userID)

		// Test content assignment to screen
		err := store.AssignContentToScreen(screen.ID, content.ID)
		assert.NoError(t, err)

		// Test getting content for screen
		screenContent, err := store.GetContentForScreen(screen.ID)
		assert.NoError(t, err)
		assert.Equal(t, content.ID, screenContent.ID)
	})

	t.Run("Screen-Playlist Assignment", func(t *testing.T) {
		// Create test user, screen, playlist, and content
		userID, _ := store.CreateUser("playlist-assignment@example.com", "password", nil)
		screen, _ := store.CreateScreen("Test Screen", nil, userID)
		playlist, _ := store.CreatePlaylist("Test Playlist", "For screen assignment", userID)
		content, _ := store.CreateContent("Test Content", "video", "https://example.com/video.mp4", 30, userID)

		// Add content to playlist
		_, _ = store.AddItemToPlaylist(playlist.ID, content.ID, 1, 30)

		// Test playlist assignment to screen
		err := store.AssignPlaylistToScreen(screen.ID, playlist.ID)
		assert.NoError(t, err)

		// Test getting playlist for screen
		screenPlaylist, err := store.GetPlaylistForScreen(screen.ID)
		assert.NoError(t, err)
		assert.Equal(t, playlist.ID, screenPlaylist.ID)

		// Test getting playlist content for screen
		playlistName, contentItems, err := store.GetPlaylistContentForScreen(screen.ID)
		assert.NoError(t, err)
		assert.Equal(t, playlist.Name, playlistName)
		assert.Equal(t, 1, len(contentItems))
		assert.Equal(t, content.URL, contentItems[0].URL)
		assert.Equal(t, 30, contentItems[0].Duration)
	})

	t.Run("Redis Integration", func(t *testing.T) {
		// Test Redis connectivity
		err := redis.Rdb.Ping(context.Background()).Err()
		assert.NoError(t, err, "Redis should be accessible")

		// Test Redis operations
		key := "test:integration:key"
		value := "test_value"

		// Set value
		err = redis.Rdb.Set(context.Background(), key, value, 0).Err()
		assert.NoError(t, err)

		// Get value
		result, err := redis.Rdb.Get(context.Background(), key).Result()
		assert.NoError(t, err)
		assert.Equal(t, value, result)

		// Cleanup
		redis.Rdb.Del(context.Background(), key)
	})
}
