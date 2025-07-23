package endpoints

import (
	"encoding/json"
	"github.com/rs/zerolog/log"
	"net/http"
	"strconv"

	"github.com/Nixie-Tech-LLC/medusa/internal/db"
	"github.com/Nixie-Tech-LLC/medusa/internal/http/api"
	"github.com/Nixie-Tech-LLC/medusa/internal/http/api/admin/packets"
	"github.com/Nixie-Tech-LLC/medusa/internal/http/middleware"
	"github.com/Nixie-Tech-LLC/medusa/internal/model"
	"github.com/gin-gonic/gin"
)

type PlaylistController struct {
	store db.Store
}

func NewPlaylistController(store db.Store) *PlaylistController {
	return &PlaylistController{store: store}
}

func RegisterPlaylistRoutes(r gin.IRoutes, store db.Store) {
	ctl := NewPlaylistController(store)

	r.GET("/playlists", api.ResolveEndpointWithAuth(ctl.listPlaylists))
	r.POST("/playlists", api.ResolveEndpointWithAuth(ctl.createPlaylist))
	r.GET("/playlists/:id", api.ResolveEndpointWithAuth(ctl.getPlaylist))
	r.PUT("/playlists/:id", api.ResolveEndpointWithAuth(ctl.updatePlaylist))
	r.DELETE("/playlists/:id", api.ResolveEndpointWithAuth(ctl.deletePlaylist))

	r.POST("/playlists/:id/items", api.ResolveEndpointWithAuth(ctl.addItem))
	r.PUT("/playlists/:id/items/:item_id", api.ResolveEndpointWithAuth(ctl.updateItem))
	r.DELETE("/playlists/:id/items/:item_id", api.ResolveEndpointWithAuth(ctl.removeItem))
	r.GET("/playlists/:id/items", api.ResolveEndpointWithAuth(ctl.listItems))
	r.PUT("/playlists/:id/items", api.ResolveEndpointWithAuth(ctl.reorderItems))
}

// listPlaylists returns all playlists created by the authenticated user.
func (p *PlaylistController) listPlaylists(ctx *gin.Context, user *model.User) (any, *api.Error) {
	all, err := p.store.ListPlaylists()
	if err != nil {
		log.Error().Err(err).Msg("[playlist] list: could not list playlists")
		return nil, &api.Error{Code: http.StatusInternalServerError, Message: "could not list playlists"}
	}

	var out []packets.PlaylistResponse
	for _, pl := range all {
		if pl.CreatedBy != user.ID {
			continue
		}
		// Items are already populated by ListPlaylists
		out = append(out, mapPlaylist(pl))
	}

	return out, nil
}

// notifyScreensPlaylistUpdated sends playlist updates to screens displaying the specified playlist
func (p *PlaylistController) notifyScreensPlaylistUpdated(playlistID int) {
	screens, err := p.store.GetScreensUsingPlaylist(playlistID)
	if err != nil {
		log.Error().Err(err).Int("playlist_id", playlistID).Msg("Failed to get screens for playlist notification")
		return
	}

	if len(screens) == 0 {
		log.Debug().Int("playlist_id", playlistID).Msg("No screens assigned to playlist")
		return
	}

	// Get updated playlist data to send to screens
	playlistName, contentItems, err := p.store.GetPlaylistContentForScreen(screens[0].ID)
	if err != nil {
		log.Error().Err(err).Int("playlist_id", playlistID).Msg("Failed to get playlist content for notification")
		return
	}

	// Create TV playlist response format
	contentList := make([]packets.TVContentItem, len(contentItems))
	for i, item := range contentItems {
		contentList[i] = packets.TVContentItem{
			URL:      item.URL,
			Duration: item.Duration,
		}
	}

	response := packets.TVPlaylistResponse{
		PlaylistName: playlistName,
		ContentList:  contentList,
	}

	messageBytes, err := json.Marshal(response)
	if err != nil {
		log.Error().Err(err).Msg("Failed to marshal playlist update message")
		return
	}

	// Send message to each screen
	for _, screen := range screens {
		if screen.DeviceID != nil {
			if err := middleware.SendMessageToScreen(*screen.DeviceID, messageBytes); err != nil {
				log.Error().Err(err).Str("device_id", *screen.DeviceID).Int("playlist_id", playlistID).Msg("Failed to send playlist update to screen")
			} else {
				log.Info().Str("device_id", *screen.DeviceID).Int("playlist_id", playlistID).Str("playlist_name", playlistName).Msg("Sent playlist update notification to screen")
			}
		}
	}
}

// createPlaylist binds and validates request, then persists a new playlist.
func (p *PlaylistController) createPlaylist(ctx *gin.Context, user *model.User) (any, *api.Error) {
	var req packets.CreatePlaylistRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Error().Err(err).Msg("[playlist] create: bad request")
		return nil, &api.Error{Code: http.StatusBadRequest, Message: err.Error()}
	}

	pl, err := p.store.CreatePlaylist(req.Name, req.Description, user.ID)
	if err != nil {
		log.Error().Err(err).Msg("[playlist] create: could not create playlist")
		return nil, &api.Error{Code: http.StatusInternalServerError, Message: "could not create playlist"}
	}

	full, _ := p.store.GetPlaylistByID(pl.ID)
	return mapPlaylist(full), nil
}

// getPlaylist fetches a single playlist by ID and checks ownership.
func (p *PlaylistController) getPlaylist(ctx *gin.Context, user *model.User) (any, *api.Error) {
	id, _ := strconv.Atoi(ctx.Param("id"))
	pl, err := p.store.GetPlaylistByID(id)
	if err != nil {
		return nil, &api.Error{Code: http.StatusNotFound, Message: "not found"}
	}
	if pl.CreatedBy != user.ID {
		return nil, &api.Error{Code: http.StatusForbidden, Message: "forbidden"}
	}
	return mapPlaylist(pl), nil
}

// updatePlaylist applies changes to an existing playlist after ownership check.
// TODO: simple test
func (p *PlaylistController) updatePlaylist(ctx *gin.Context, user *model.User) (any, *api.Error) {
	id, _ := strconv.Atoi(ctx.Param("id"))
	existing, err := p.store.GetPlaylistByID(id)
	if err != nil || existing.CreatedBy != user.ID {
		return nil, &api.Error{Code: http.StatusForbidden, Message: "forbidden"}
	}

	var req packets.UpdatePlaylistRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return nil, &api.Error{Code: http.StatusBadRequest, Message: err.Error()}
	}

	if err := p.store.UpdatePlaylist(id, req.Name, req.Description); err != nil {
		return nil, &api.Error{Code: http.StatusInternalServerError, Message: err.Error()}
	}

	// Notify screens about playlist update
	go p.notifyScreensPlaylistUpdated(id)

	// return updated playlist
	full, _ := p.store.GetPlaylistByID(id)
	return mapPlaylist(full), nil
}

// deletePlaylist removes a playlist after verifying user ownership.
// TODO: simple test
func (p *PlaylistController) deletePlaylist(ctx *gin.Context, user *model.User) (any, *api.Error) {
	id, _ := strconv.Atoi(ctx.Param("id"))
	pl, err := p.store.GetPlaylistByID(id)
	if err != nil || pl.CreatedBy != user.ID {
		return nil, &api.Error{Code: http.StatusForbidden, Message: "forbidden"}
	}

	// Notify screens before deletion
	go p.notifyScreensPlaylistUpdated(id)

	if err := p.store.DeletePlaylist(id); err != nil {
		return nil, &api.Error{Code: http.StatusInternalServerError, Message: err.Error()}
	}
	return nil, nil
}

// addItem inserts a new item into a playlist at the specified position.
// TODO: simple test
func (p *PlaylistController) addItem(ctx *gin.Context, user *model.User) (any, *api.Error) {
	pid, _ := strconv.Atoi(ctx.Param("id"))
	pl, err := p.store.GetPlaylistByID(pid)
	if err != nil || pl.CreatedBy != user.ID {
		return nil, &api.Error{Code: http.StatusForbidden, Message: "forbidden"}
	}

	var req packets.AddPlaylistItemRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return nil, &api.Error{Code: http.StatusBadRequest, Message: err.Error()}
	}

	// decide duration
	defaultDur := 5
	if req.Duration != nil {
		defaultDur = *req.Duration
	}

	// 1) fetch existing items so we can compute the next position
	existingItems, err := p.store.ListPlaylistItems(pid)
	if err != nil {
		log.Printf("[playlist] list items: %v", err)
		return nil, &api.Error{Code: http.StatusInternalServerError, Message: "could not list playlist items"}
	}

	// 2) compute new position = (last position) + 1
	nextPos := 1
	if len(existingItems) > 0 {
		nextPos = existingItems[len(existingItems)-1].Position + 1
	}

	// 3) insert at end
	item, err := p.store.AddItemToPlaylist(pid, req.ContentID, nextPos, defaultDur)
	if err != nil {
		log.Printf("[playlist] add item: %v", err)
		return nil, &api.Error{Code: http.StatusInternalServerError, Message: "could not add item"}
	}

	// Notify screens about playlist update
	go p.notifyScreensPlaylistUpdated(pid)

	return mapItem(item), nil
}

// updateItem changes position or duration of an existing playlist item
// TODO: simple test
func (p *PlaylistController) updateItem(ctx *gin.Context, user *model.User) (any, *api.Error) {
	pid, _ := strconv.Atoi(ctx.Param("id"))
	pl, _ := p.store.GetPlaylistByID(pid)
	if pl.CreatedBy != user.ID {
		return nil, &api.Error{Code: http.StatusForbidden, Message: "forbidden"}
	}
	id, _ := strconv.Atoi(ctx.Param("item_id"))
	var req packets.UpdatePlaylistItemRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return nil, &api.Error{Code: http.StatusBadRequest, Message: err.Error()}
	}
	if err := p.store.UpdatePlaylistItem(id, req.Position, req.Duration); err != nil {
		return nil, &api.Error{Code: http.StatusInternalServerError, Message: err.Error()}
	}

	// Notify screens about playlist update
	go p.notifyScreensPlaylistUpdated(pid)

	return nil, nil
}

// removeItem deletes an item from a playlist.
func (p *PlaylistController) removeItem(ctx *gin.Context, user *model.User) (any, *api.Error) {
	pid, _ := strconv.Atoi(ctx.Param("id"))
	pl, _ := p.store.GetPlaylistByID(pid)
	if pl.CreatedBy != user.ID {
		return nil, &api.Error{Code: http.StatusForbidden, Message: "forbidden"}
	}

	iid, _ := strconv.Atoi(ctx.Param("item_id"))
	if err := p.store.RemovePlaylistItem(iid); err != nil {
		return nil, &api.Error{Code: http.StatusInternalServerError, Message: err.Error()}
	}

	// Notify screens about playlist update
	go p.notifyScreensPlaylistUpdated(pid)

	return nil, nil
}

// listItems returns all items in a playlist (with ownership check)
func (p *PlaylistController) listItems(ctx *gin.Context, user *model.User) (any, *api.Error) {
	pid, _ := strconv.Atoi(ctx.Param("id"))
	pl, err := p.store.GetPlaylistByID(pid)
	if err != nil || pl.CreatedBy != user.ID {
		return nil, &api.Error{Code: http.StatusForbidden, Message: "forbidden"}
	}

	items, err := p.store.ListPlaylistItems(pid)
	if err != nil {
		log.Printf("[playlist] list items: %v", err)
		return nil, &api.Error{Code: http.StatusInternalServerError, Message: "could not list playlist items"}
	}

	out := make([]packets.PlaylistItemResponse, len(items))
	for i, it := range items {
		out[i] = mapItem(it)
	}
	return out, nil
}

// reorderItems takes a JSON array of item IDs in the new order,
// updates their position (1-based) in a single transaction,
// and returns the freshly-ordered list.
func (p *PlaylistController) reorderItems(ctx *gin.Context, user *model.User) (any, *api.Error) {
	pid, _ := strconv.Atoi(ctx.Param("id"))
	pl, err := p.store.GetPlaylistByID(pid)
	if err != nil || pl.CreatedBy != user.ID {
		return nil, &api.Error{Code: http.StatusForbidden, Message: "forbidden"}
	}

	var req struct {
		ItemIDs []int `json:"item_ids" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return nil, &api.Error{Code: http.StatusBadRequest, Message: err.Error()}
	}

	if err := p.store.ReorderPlaylistItems(pid, req.ItemIDs); err != nil {
		log.Printf("[playlist] reorder: %v", err)
		return nil, &api.Error{Code: http.StatusInternalServerError, Message: "could not reorder items"}
	}

	// Notify screens about playlist update
	go p.notifyScreensPlaylistUpdated(pid)

	// return the newly-ordered list
	return p.listItems(ctx, user)
}

func mapPlaylist(pl model.Playlist) packets.PlaylistResponse {
	items := make([]packets.PlaylistItemResponse, len(pl.Items))
	log.Printf("[playlists] mapPlaylists %v", pl.Items)
	for i, it := range pl.Items {
		items[i] = mapItem(it)
	}
	return packets.PlaylistResponse{
		ID:          pl.ID,
		Name:        pl.Name,
		Description: *pl.Description,
		CreatedBy:   pl.CreatedBy,
		CreatedAt:   pl.CreatedAt,
		UpdatedAt:   pl.UpdatedAt,
		Items:       items,
	}
}

// mapItem transforms a model.PlaylistItem into the API response packet.
func mapItem(it model.PlaylistItem) packets.PlaylistItemResponse {
	return packets.PlaylistItemResponse{
		ID:        it.ID,
		ContentID: it.ContentID,
		Position:  it.Position,
		Duration:  *it.Duration,
		CreatedAt: it.CreatedAt,
	}
}
