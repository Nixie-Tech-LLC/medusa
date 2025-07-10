package endpoints

import (
    "net/http"
    "strconv"
    "log"
	"github.com/rs/zerolog"

    "github.com/gin-gonic/gin"
    "github.com/Nixie-Tech-LLC/medusa/internal/db"
    "github.com/Nixie-Tech-LLC/medusa/internal/http/api"
    "github.com/Nixie-Tech-LLC/medusa/internal/http/api/admin/packets"
    "github.com/Nixie-Tech-LLC/medusa/internal/model"
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
}

// listPlaylists returns all playlists created by the authenticated user.
func (p *PlaylistController) listPlaylists(ctx *gin.Context, user *model.User) (any, *api.Error) {
    all, err := p.store.ListPlaylists()
    if err != nil {
        log.Printf("[playlist] list: %v", err)
        return nil, &api.Error{Code: http.StatusInternalServerError, Message: "could not list playlists"}
    }

    var out []packets.PlaylistResponse
    for _, pl := range all {
        if pl.CreatedBy != user.ID {
            continue // skip playlists not owned by user
        }
        out = append(out, mapPlaylist(pl))
    }
    return out, nil
}

// createPlaylist binds and validates request, then persists a new playlist.
func (p *PlaylistController) createPlaylist(ctx *gin.Context, user *model.User) (any, *api.Error) {
    var req packets.CreatePlaylistRequest
    if err := ctx.ShouldBindJSON(&req); err != nil {
        return nil, &api.Error{Code: http.StatusBadRequest, Message: err.Error()}
    }

    pl, err := p.store.CreatePlaylist(req.Name, req.Description, user.ID)
    if err != nil {
        log.Printf("[playlist] create: %v", err)
        return nil, &api.Error{Code: http.StatusInternalServerError, Message: "could not create playlist"}
    }

    // reload to include items (initially empty)
    full, _ := p.store.GetPlaylistByID(pl.ID)
	return mapPlaylist(full), log.Info().Str("username", user.Name)
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

    // use provided duration or fallback to first item's duration
    defaultDur := *pl.Items[0].Duration
    if req.Duration != nil {
        defaultDur = *req.Duration
    }
    item, err := p.store.AddItemToPlaylist(pid, req.ContentID, req.Position, defaultDur)
    if err != nil {
        log.Printf("[playlist] add item: %v", err)
        return nil, &api.Error{Code: http.StatusInternalServerError, Message: "could not add item"}
    }
    return mapItem(item), nil
}

// updateItem changes position or duration of an existing playlist item.
// TODO: simple test
func (p *PlaylistController) updateItem(ctx *gin.Context, user *model.User) (any, *api.Error) {
    pid, _ := strconv.Atoi(ctx.Param("id"))
    pl, _ := p.store.GetPlaylistByID(pid)
    if pl.CreatedBy != user.ID {
        return nil, &api.Error{Code: http.StatusForbidden, Message: "forbidden"}
    }

    iid, _ := strconv.Atoi(ctx.Param("item_id"))
    var req packets.UpdatePlaylistItemRequest
    if err := ctx.ShouldBindJSON(&req); err != nil {
        return nil, &api.Error{Code: http.StatusBadRequest, Message: err.Error()}
    }
    if err := p.store.UpdatePlaylistItem(iid, req.Position, req.Duration); err != nil {
        return nil, &api.Error{Code: http.StatusInternalServerError, Message: err.Error()}
    }
    return nil, nil
}

// removeItem deletes an item from a playlist.
// TODO: simple test
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
    return nil, nil
}

// mapPlaylist transforms a model.Playlist into the API response packet.
func mapPlaylist(pl model.Playlist) packets.PlaylistResponse {
    items := make([]packets.PlaylistItemResponse, len(pl.Items))
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

