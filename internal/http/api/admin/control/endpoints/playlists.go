package endpoints

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/Nixie-Tech-LLC/medusa/internal/db"
	"github.com/Nixie-Tech-LLC/medusa/internal/http/api"
	"github.com/Nixie-Tech-LLC/medusa/internal/http/api/admin/control/packets"
	"github.com/Nixie-Tech-LLC/medusa/internal/http/api/admin/control/utils"
	"github.com/Nixie-Tech-LLC/medusa/internal/model"
	"github.com/Nixie-Tech-LLC/medusa/internal/redis"
)


type PlaylistController struct {
	store db.Store
}

func newPlaylistController(store db.Store) *PlaylistController {
	return &PlaylistController{store: store}
}

// PlaylistModule mounts all authenticated /playlists endpoints.
func PlaylistModule(store db.Store) api.Module {
	ctl := newPlaylistController(store)
	return api.ModuleFunc(func(c *api.Controller) {
		c.GET("/playlists", 		ctl.listPlaylists)
		c.POST("/playlists", 		ctl.createPlaylist)
		c.GET("/playlists/:id", 	ctl.getPlaylist)
		c.PUT("/playlists/:id", 	ctl.updatePlaylist)
		c.DELETE("/playlists/:id", 	ctl.deletePlaylist)

		c.POST("/playlists/:id/items", 				ctl.addItem)
		c.PUT("/playlists/:id/items/:item_id", 		ctl.updateItem)
		c.DELETE("/playlists/:id/items/:item_id", 	ctl.removeItem)
		c.GET("/playlists/:id/items",		 		ctl.listItems)
		c.PUT("/playlists/:id/items", 				ctl.reorderItems)

		c.POST("/playlists/:id/integrations", ctl.addIntegration)
	})
}

func (p *PlaylistController) notifyScreensPlaylistUpdated(playlistID int) {
	screens, err := p.store.GetScreensUsingPlaylist(playlistID)
	if err != nil {
		log.Error().Err(err).Int("playlist_id", playlistID).
			Msg("failed to get screens for playlist notification")
		return
	}

	if len(screens) == 0 {
		log.Debug().Int("playlist_id", playlistID).Msg("no screens assigned to playlist")
		return
	}

	etagKey := fmt.Sprintf("playlist:%d:etag", playlistID)
	if err := redis.Rdb.Del(context.Background(), etagKey).Err(); err != nil {
		log.Warn().Err(err).Int("playlist_id", playlistID).Str("etag_key", etagKey).
			Msg("failed to invalidate playlist ETag cache")
	} else {
		log.Debug().Int("playlist_id", playlistID).Str("etag_key", etagKey).
			Msg("invalidated playlist ETag cache")
	}

	log.Info().Int("playlist_id", playlistID).Int("affected_screens", len(screens)).
		Msg("playlist updated - invalidated playlist ETag for all affected screens")
}

func mapPlaylist(pl model.Playlist) packets.PlaylistResponse {
	items := make([]packets.PlaylistItemResponse, len(pl.Items))
	log.Debug().Int("items_count", len(pl.Items)).Msg("[playlists] mapPlaylist")

	for i, it := range pl.Items {
		items[i] = mapItem(it)
	}

	var desc string
	if pl.Description != nil {
		desc = *pl.Description
	}

	return packets.PlaylistResponse{
		ID:          pl.ID,
		Name:        pl.Name,
		Description: desc,
		CreatedBy:   pl.CreatedBy,
		CreatedAt:   pl.CreatedAt,
		UpdatedAt:   pl.UpdatedAt,
		Items:       items,
	}
}

func mapItem(it model.PlaylistItem) packets.PlaylistItemResponse {
	return packets.PlaylistItemResponse{
		ID:        it.ID,
		ContentID: it.ContentID,
		Position:  it.Position,
		Duration:  it.Duration,
		CreatedAt: it.CreatedAt,
	}
}

// ===== Handlers (AuthHandlerFunc signatures) =====

func (p *PlaylistController) listPlaylists(ctx *gin.Context, user *model.User) (any, *api.APIError) {
	all, err := p.store.ListPlaylists()
	if err != nil {
		log.Error().Err(err).Msg("[playlist] list: could not list playlists")
		return nil, &api.APIError{Code: http.StatusInternalServerError, Message: "could not list playlists"}
	}

	var out []packets.PlaylistResponse
	for _, pl := range all {
		if pl.CreatedBy != user.ID {
			continue
		}
		out = append(out, mapPlaylist(pl))
	}
	return out, nil
}

func (p *PlaylistController) createPlaylist(ctx *gin.Context, user *model.User) (any, *api.APIError) {
	var req packets.CreatePlaylistRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Error().Err(err).Msg("[playlist] create: bad request")
		return nil, &api.APIError{Code: http.StatusBadRequest, Message: err.Error()}
	}

	pl, err := p.store.CreatePlaylist(req.Name, req.Description, user.ID)
	if err != nil {
		log.Error().Err(err).Msg("[playlist] create: could not create playlist")
		return nil, &api.APIError{Code: http.StatusInternalServerError, Message: "could not create playlist"}
	}

	full, _ := p.store.GetPlaylistByID(pl.ID)
	return mapPlaylist(full), nil
}

func (p *PlaylistController) getPlaylist(ctx *gin.Context, user *model.User) (any, *api.APIError) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		return nil, &api.APIError{Code: http.StatusBadRequest, Message: "invalid id"}
	}
	pl, err := p.store.GetPlaylistByID(id)
	if err != nil {
		return nil, &api.APIError{Code: http.StatusNotFound, Message: "not found"}
	}
	if pl.CreatedBy != user.ID {
		return nil, &api.APIError{Code: http.StatusForbidden, Message: "forbidden"}
	}
	return mapPlaylist(pl), nil
}

func (p *PlaylistController) updatePlaylist(ctx *gin.Context, user *model.User) (any, *api.APIError) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		return nil, &api.APIError{Code: http.StatusBadRequest, Message: "invalid id"}
	}

	existing, err := p.store.GetPlaylistByID(id)
	if err != nil || existing.CreatedBy != user.ID {
		return nil, &api.APIError{Code: http.StatusForbidden, Message: "forbidden"}
	}

	var req packets.UpdatePlaylistRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return nil, &api.APIError{Code: http.StatusBadRequest, Message: err.Error()}
	}

	if err := p.store.UpdatePlaylist(id, req.Name, req.Description); err != nil {
		return nil, &api.APIError{Code: http.StatusInternalServerError, Message: err.Error()}
	}

	go p.notifyScreensPlaylistUpdated(id)

	full, _ := p.store.GetPlaylistByID(id)
	return mapPlaylist(full), nil
}

func (p *PlaylistController) deletePlaylist(ctx *gin.Context, user *model.User) (any, *api.APIError) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		return nil, &api.APIError{Code: http.StatusBadRequest, Message: "invalid id"}
	}

	pl, err := p.store.GetPlaylistByID(id)
	if err != nil || pl.CreatedBy != user.ID {
		return nil, &api.APIError{Code: http.StatusForbidden, Message: "forbidden"}
	}

	// Notify screens before deletion
	go p.notifyScreensPlaylistUpdated(id)

	if err := p.store.DeletePlaylist(id); err != nil {
		return nil, &api.APIError{Code: http.StatusInternalServerError, Message: err.Error()}
	}
	return nil, nil
}

func (p *PlaylistController) addItem(ctx *gin.Context, user *model.User) (any, *api.APIError) {
	pid, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		return nil, &api.APIError{Code: http.StatusBadRequest, Message: "invalid playlist id"}
	}

	pl, err := p.store.GetPlaylistByID(pid)
	if err != nil || pl.CreatedBy != user.ID {
		return nil, &api.APIError{Code: http.StatusForbidden, Message: "forbidden"}
	}

	var req packets.AddPlaylistItemRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return nil, &api.APIError{Code: http.StatusBadRequest, Message: err.Error()}
	}

	existingItems, err := p.store.ListPlaylistItems(pid)
	if err != nil {
		log.Error().Err(err).Msg("[playlist] list items failed")
		return nil, &api.APIError{Code: http.StatusInternalServerError, Message: "could not list playlist items"}
	}

	nextPos := 1
	if len(existingItems) > 0 {
		nextPos = existingItems[len(existingItems)-1].Position + 1
	}

	duration := req.Duration
	item, err := p.store.AddItemToPlaylist(pid, req.ContentID, nextPos, duration)
	if err != nil {
		log.Error().Err(err).Msg("[playlist] add item failed")
		return nil, &api.APIError{Code: http.StatusInternalServerError, Message: "could not add item"}
	}

	go p.notifyScreensPlaylistUpdated(pid)
	return mapItem(item), nil
}

func (p *PlaylistController) updateItem(ctx *gin.Context, user *model.User) (any, *api.APIError) {
	pid, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		return nil, &api.APIError{Code: http.StatusBadRequest, Message: "invalid playlist id"}
	}

	pl, err := p.store.GetPlaylistByID(pid)
	if err != nil || pl.CreatedBy != user.ID {
		return nil, &api.APIError{Code: http.StatusForbidden, Message: "forbidden"}
	}

	id, err := strconv.Atoi(ctx.Param("item_id"))
	if err != nil {
		return nil, &api.APIError{Code: http.StatusBadRequest, Message: "invalid item id"}
	}

	var req packets.UpdatePlaylistItemRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return nil, &api.APIError{Code: http.StatusBadRequest, Message: err.Error()}
	}

	if err := p.store.UpdatePlaylistItem(id, req.Position, req.Duration); err != nil {
		return nil, &api.APIError{Code: http.StatusInternalServerError, Message: err.Error()}
	}

	go p.notifyScreensPlaylistUpdated(pid)
	return nil, nil
}

func (p *PlaylistController) removeItem(ctx *gin.Context, user *model.User) (any, *api.APIError) {
	pid, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		return nil, &api.APIError{Code: http.StatusBadRequest, Message: "invalid playlist id"}
	}

	pl, err := p.store.GetPlaylistByID(pid)
	if err != nil || pl.CreatedBy != user.ID {
		return nil, &api.APIError{Code: http.StatusForbidden, Message: "forbidden"}
	}

	iid, err := strconv.Atoi(ctx.Param("item_id"))
	if err != nil {
		return nil, &api.APIError{Code: http.StatusBadRequest, Message: "invalid item id"}
	}

	if err := p.store.RemovePlaylistItem(iid); err != nil {
		return nil, &api.APIError{Code: http.StatusInternalServerError, Message: err.Error()}
	}

	go p.notifyScreensPlaylistUpdated(pid)
	return nil, nil
}

func (p *PlaylistController) listItems(ctx *gin.Context, user *model.User) (any, *api.APIError) {
	pid, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		return nil, &api.APIError{Code: http.StatusBadRequest, Message: "invalid playlist id"}
	}

	pl, err := p.store.GetPlaylistByID(pid)
	if err != nil || pl.CreatedBy != user.ID {
		return nil, &api.APIError{Code: http.StatusForbidden, Message: "forbidden"}
	}

	items, err := p.store.ListPlaylistItems(pid)
	if err != nil {
		log.Error().Err(err).Msg("[playlist] list items failed")
		return nil, &api.APIError{Code: http.StatusInternalServerError, Message: "could not list playlist items"}
	}

	out := make([]packets.PlaylistItemResponse, len(items))
	for i, it := range items {
		out[i] = mapItem(it)
	}
	return out, nil
}

func (p *PlaylistController) reorderItems(ctx *gin.Context, user *model.User) (any, *api.APIError) {
	pid, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		return nil, &api.APIError{Code: http.StatusBadRequest, Message: "invalid playlist id"}
	}

	pl, err := p.store.GetPlaylistByID(pid)
	if err != nil || pl.CreatedBy != user.ID {
		return nil, &api.APIError{Code: http.StatusForbidden, Message: "forbidden"}
	}

	var req packets.ReorderItemsRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		return nil, &api.APIError{Code: http.StatusBadRequest, Message: err.Error()}
	}

	if err := p.store.ReorderPlaylistItems(pid, req.ItemIDs); err != nil {
		log.Error().Err(err).Msg("[playlist] reorder failed")
		return nil, &api.APIError{Code: http.StatusInternalServerError, Message: "could not reorder items"}
	}

	go p.notifyScreensPlaylistUpdated(pid)
	return p.listItems(ctx, user)
}

func (p *PlaylistController) addIntegration(ctx *gin.Context, user *model.User) (any, *api.APIError) {
	pid, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		return nil, &api.APIError{Code: http.StatusBadRequest, Message: "invalid playlist id"}
	}

	pl, err := p.store.GetPlaylistByID(pid)
	if err != nil || pl.CreatedBy != user.ID {
		return nil, &api.APIError{Code: http.StatusForbidden, Message: "forbidden"}
	}

	var req packets.AddIntegrationRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		return nil, &api.APIError{Code: http.StatusBadRequest, Message: err.Error()}
	}

	var url string
	switch req.IntegrationName {
		case "athan":
			conUrl, apiErr := utils.SetupAthan(req.Config)
			if apiErr != nil {
				return nil, apiErr
			}
			url = conUrl
		default:
			return nil, &api.APIError{Code: http.StatusBadRequest, Message: "unknown integration"}
	}

	items, err := p.store.ListPlaylistItems(pid)
	if err != nil {
		return nil, &api.APIError{Code: http.StatusInternalServerError, Message: "could not list playlist items"}
	}

	for _, it := range items {
		content, err := p.store.GetContentByID(it.ContentID)
		if err != nil {
			continue
		}
		if content.URL == url {
			return mapItem(it), nil
		}
	}

	dur := 10
	if req.Duration != nil {
		dur = *req.Duration
	}

	content, err := p.store.CreateContent(
		req.IntegrationName, // name
		"text/html",         // type
		url,                 // URL
		1920,                // width
		1080,                // height
		user.ID,
	)
	if err != nil {
		log.Error().Err(err).Msg("create integration content failed")
		return nil, &api.APIError{Code: http.StatusInternalServerError, Message: "could not create content"}
	}

	// Determine position
	var pos int
	if req.Position != nil {
		pos = *req.Position
		if pos < 1 {
			pos = 1
		} else if pos > len(items)+1 {
			pos = len(items) + 1
		}

		// Shift existing items if inserting in the middle
		if pos <= len(items) {
			// Need to shift items at position >= pos
			for _, item := range items {
				if item.Position >= pos {
					newPos := pos + 1
					if err := p.store.UpdatePlaylistItem(item.ID, &newPos, &item.Duration); err != nil {
						log.Error().Err(err).Msg("Failed to shift playlist item position")
						return nil, &api.APIError{Code: http.StatusInternalServerError, Message: "could not reorder items"}
					}
				}
			}
		}
	} else {
		// Default to appending at the end
		if len(items) > 0 {
			pos = items[len(items)-1].Position + 1
		} else {
			pos = 1
		}
	}

	item, err := p.store.AddItemToPlaylist(pid, content.ID, pos, dur)
	if err != nil {
		log.Error().Err(err).Msg("add integration item failed")
		return nil, &api.APIError{Code: http.StatusInternalServerError, Message: "could not add item"}
	}

	go p.notifyScreensPlaylistUpdated(pid)
	return mapItem(item), nil
}

