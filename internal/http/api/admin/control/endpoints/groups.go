package endpoints

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Nixie-Tech-LLC/medusa/internal/db"
	"github.com/Nixie-Tech-LLC/medusa/internal/http/api"
	"github.com/Nixie-Tech-LLC/medusa/internal/http/api/admin/control/packets"
	"github.com/Nixie-Tech-LLC/medusa/internal/model"
)

type GroupController struct{ store db.Store }

func newGroupController(store db.Store) *GroupController { return &GroupController{store: store} 
func newGroupController(store db.Store) *GroupController { return &GroupController{store: store} }

// In module registration (e.g., Admin control module)
func ScreenGroupModule(store db.Store) api.Module {
	ctl := newGroupController(store)
	return api.ModuleFunc(func(c *api.Controller) {
		c.GET("/screen-groups", ctl.listGroups)
		c.POST("/screen-groups", ctl.createGroup)
		c.PUT("/screen-groups/:id", ctl.renameGroup)
		c.DELETE("/screen-groups/:id", ctl.deleteGroup)
		c.DELETE("/screen-groups/:id",            ctl.deleteGroup)

		c.GET("/screen-groups/:id/screens", ctl.listScreensInGroup)
		c.POST("/screen-groups/:id/screens", ctl.addScreenToGroup) // body: {screen_id}
		c.POST("/screen-groups/:id/screens",      ctl.addScreenToGroup)     // body: {screen_id}
		c.DELETE("/screen-groups/:id/screens/:sid", ctl.removeScreenFromGroup)

		c.GET("/screens/:id/groups", ctl.listGroupsForScreen)
		c.GET("/screens/:id/groups",              ctl.listGroupsForScreen)
	})
}

// GET /api/admin/screen-groups
func (g *GroupController) listGroups(ctx *gin.Context, user *model.User) (any, *api.APIError) {
	if err != nil {
		return nil, &api.APIError{Code: http.StatusInternalServerError, Message: err.Error()}
	}
	if err != nil { return nil, &api.APIError{Code: http.StatusInternalServerError, Message: err.Error()} }
	out := make([]packets.ScreenGroupResponse, 0, len(groups))
	for _, gr := range groups {
		out = append(out, packets.ScreenGroupResponse{
			ID: gr.ID, Name: gr.Name, Description: gr.Description,
			CreatedAt: gr.CreatedAt.Format(time.RFC3339),
			UpdatedAt: gr.UpdatedAt.Format(time.RFC3339),
		})
	}
	return out, nil
}

// POST /api/admin/screen-groups
func (g *GroupController) createGroup(ctx *gin.Context, user *model.User) (any, *api.APIError) {
	var req packets.CreateScreenGroupRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return nil, &api.APIError{Code: http.StatusBadRequest, Message: err.Error()}
	}
	name := req.Name
	grp, err := g.store.CreateScreenGroup(user.ID, &name, req.Description)
	if err != nil {
		return nil, &api.APIError{Code: http.StatusConflict, Message: err.Error()} // unique name per user
	}
	return packets.ScreenGroupResponse{
		ID: grp.ID, Name: grp.Name, Description: grp.Description,
		CreatedAt: grp.CreatedAt.Format(time.RFC3339),
		UpdatedAt: grp.UpdatedAt.Format(time.RFC3339),
	}, nil
}

// PUT /api/admin/screen-groups/:id
func (g *GroupController) renameGroup(ctx *gin.Context, user *model.User) (any, *api.APIError) {
	if err != nil {
		return nil, &api.APIError{Code: http.StatusBadRequest, Message: "invalid id"}
	}
	if err != nil { return nil, &api.APIError{Code: http.StatusBadRequest, Message: "invalid id"} }
	var req packets.RenameScreenGroupRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return nil, &api.APIError{Code: http.StatusBadRequest, Message: err.Error()}
	}
	if err != nil {
		return nil, &api.APIError{Code: http.StatusNotFound, Message: "group not found"}
	}
	if err != nil { return nil, &api.APIError{Code: http.StatusNotFound, Message: "group not found"} }
	return packets.ScreenGroupResponse{
		ID: grp.ID, Name: grp.Name, Description: grp.Description,
		CreatedAt: grp.CreatedAt.Format(time.RFC3339),
		UpdatedAt: grp.UpdatedAt.Format(time.RFC3339),
	}, nil
}

// DELETE /api/admin/screen-groups/:id
func (g *GroupController) deleteGroup(ctx *gin.Context, user *model.User) (any, *api.APIError) {
	if err != nil {
		return nil, &api.APIError{Code: http.StatusBadRequest, Message: "invalid id"}
	}
	if err != nil { return nil, &api.APIError{Code: http.StatusBadRequest, Message: "invalid id"} }
	if err := g.store.DeleteScreenGroup(user.ID, id); err != nil {
		return nil, &api.APIError{Code: http.StatusNotFound, Message: "group not found"}
	}
	return gin.H{"deleted": true}, nil
}

// GET /api/admin/screen-groups/:id/screens
func (g *GroupController) listScreensInGroup(ctx *gin.Context, user *model.User) (any, *api.APIError) {
	if err != nil {
		return nil, &api.APIError{Code: http.StatusBadRequest, Message: "invalid id"}
	}
	if err != nil { return nil, &api.APIError{Code: http.StatusBadRequest, Message: "invalid id"} }
	if err != nil {
		return nil, &api.APIError{Code: http.StatusNotFound, Message: "group not found"}
	}
	if err != nil { return nil, &api.APIError{Code: http.StatusNotFound, Message: "group not found"} }
	resp := make([]packets.ScreenResponse, 0, len(scr))
	for _, s := range scr {
		resp = append(resp, packets.ScreenResponse{
			ID: s.ID, DeviceID: s.DeviceID, ClientInformation: s.ClientInformation,
			ClientWidth: s.ClientWidth, ClientHeight: s.ClientHeight, Name: s.Name,
			Location: s.Location, Paired: s.Paired,
			CreatedAt: s.CreatedAt.Format(time.RFC3339),
			UpdatedAt: s.UpdatedAt.Format(time.RFC3339),
		})
	}
	return resp, nil
}

// POST /api/admin/screen-groups/:id/screens
func (g *GroupController) addScreenToGroup(ctx *gin.Context, user *model.User) (any, *api.APIError) {
	if err != nil {
		return nil, &api.APIError{Code: http.StatusBadRequest, Message: "invalid id"}
	}
	if err != nil { return nil, &api.APIError{Code: http.StatusBadRequest, Message: "invalid id"} }
	var req packets.ModifyGroupMembershipRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return nil, &api.APIError{Code: http.StatusBadRequest, Message: err.Error()}
	}
	if err := g.store.AddScreenToGroup(user.ID, id, req.ScreenID); err != nil {
		return nil, &api.APIError{Code: http.StatusForbidden, Message: err.Error()}
	}
	return gin.H{"added": true}, nil
}

// DELETE /api/admin/screen-groups/:id/screens/:sid
func (g *GroupController) removeScreenFromGroup(ctx *gin.Context, user *model.User) (any, *api.APIError) {
	if err != nil {
		return nil, &api.APIError{Code: http.StatusBadRequest, Message: "invalid group id"}
	}
	if err != nil { return nil, &api.APIError{Code: http.StatusBadRequest, Message: "invalid group id"} }
	if err != nil {
		return nil, &api.APIError{Code: http.StatusBadRequest, Message: "invalid screen id"}
	}
	if err != nil { return nil, &api.APIError{Code: http.StatusBadRequest, Message: "invalid screen id"} }
	if err := g.store.RemoveScreenFromGroup(user.ID, gid, sid); err != nil {
		return nil, &api.APIError{Code: http.StatusForbidden, Message: err.Error()}
	}
	return gin.H{"removed": true}, nil
}

// GET /api/admin/screens/:id/groups (optional helper)
func (g *GroupController) listGroupsForScreen(ctx *gin.Context, user *model.User) (any, *api.APIError) {
	if err != nil {
		return nil, &api.APIError{Code: http.StatusBadRequest, Message: "invalid id"}
	}
	if err != nil { return nil, &api.APIError{Code: http.StatusBadRequest, Message: "invalid id"} }
	if err != nil {
		return nil, &api.APIError{Code: http.StatusInternalServerError, Message: err.Error()}
	}
	if err != nil { return nil, &api.APIError{Code: http.StatusInternalServerError, Message: err.Error()} }
	out := make([]packets.ScreenGroupResponse, 0, len(groups))
	for _, gr := range groups {
		out = append(out, packets.ScreenGroupResponse{
			ID: gr.ID, Name: gr.Name, Description: gr.Description,
			CreatedAt: gr.CreatedAt.Format(time.RFC3339),
			UpdatedAt: gr.UpdatedAt.Format(time.RFC3339),
		})
	}
	return out, nil
}
