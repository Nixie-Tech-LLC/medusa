package endpoints

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/Nixie-Tech-LLC/medusa/internal/db"
	"github.com/Nixie-Tech-LLC/medusa/internal/http/api"
	"github.com/Nixie-Tech-LLC/medusa/internal/http/api/admin/control/packets"
	"github.com/Nixie-Tech-LLC/medusa/internal/model"
)

type ScheduleController struct {
	store db.Store
}

func NewScheduleController(store db.Store) *ScheduleController {
	return &ScheduleController{store: store}
}

func ScheduleModule(store db.Store) api.Module {
	ctl := NewScheduleController(store)
	return api.ModuleFunc(func(c *api.Controller) {
		// top-level schedules
		c.GET("/schedules", ctl.listSchedules)
		c.POST("/schedules", ctl.createSchedule)
		c.DELETE("/schedules/:id", ctl.deleteSchedule)

		// schedule <-> screen
		c.POST("/schedules/:id/screens", ctl.assignScheduleToScreen)
		c.DELETE("/schedules/:id/screens/:screen_id", ctl.unassignScheduleFromScreen)

		// windows (playlist assignments)
		c.POST("/schedules/:id/windows", ctl.createWindow)
		c.DELETE("/schedules/windows/:window_id", ctl.deleteWindow)

		// calendar feed for GUI (expand occurrences between [from,to))
		c.GET("/schedules/:id/occurrences", ctl.listOccurrences)
	})
}

func (s *ScheduleController) listSchedules(ctx *gin.Context, user *model.User) (any, *api.APIError) {
	list, err := s.store.ListSchedules(user.ID)
	if err != nil {
		return nil, &api.APIError{Code: http.StatusInternalServerError, Message: "failed to list schedules"}
	}

	response := make([]packets.ScheduleResponse, 0, len(list))
	for _, it := range list {
		response = append(response, packets.ScheduleResponse{
			ID:        it.ID,
			Name:      it.Name,
			CreatedAt: it.CreatedAt.Format(time.RFC3339),
			UpdatedAt: it.UpdatedAt.Format(time.RFC3339),
		})
	}
	return response, nil
}

func (s *ScheduleController) createSchedule(ctx *gin.Context, user *model.User) (any, *api.APIError) {
	var request packets.CreateScheduleRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		return nil, &api.APIError{Code: http.StatusBadRequest, Message: err.Error()}
	}

	sc, err := s.store.CreateSchedule(request.Name, user.ID)
	if err != nil {
		return nil, &api.APIError{Code: http.StatusInternalServerError, Message: "could not create schedule"}
	}

	response := packets.ScheduleResponse{
		ID:        sc.ID,
		Name:      sc.Name,
		CreatedAt: sc.CreatedAt.Format(time.RFC3339),
		UpdatedAt: sc.UpdatedAt.Format(time.RFC3339),
	}
	return response, nil
}

func (s *ScheduleController) deleteSchedule(ctx *gin.Context, user *model.User) (any, *api.APIError) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		return nil, &api.APIError{Code: http.StatusBadRequest, Message: "invalid id"}
	}

	owned, err := s.store.GetSchedule(id)
	if err != nil {
		return nil, &api.APIError{Code: http.StatusNotFound, Message: "schedule not found"}
	}
	if owned.CreatedBy != user.ID {
		return nil, &api.APIError{Code: http.StatusForbidden, Message: "forbidden"}
	}

	if err := s.store.DeleteSchedule(id); err != nil {
		return nil, &api.APIError{Code: http.StatusInternalServerError, Message: "could not delete schedule"}
	}

	response := gin.H{"message": "deleted"}
	return response, nil
}

func (s *ScheduleController) assignScheduleToScreen(ctx *gin.Context, user *model.User) (any, *api.APIError) {
	scheduleID, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		return nil, &api.APIError{Code: http.StatusBadRequest, Message: "invalid schedule id"}
	}

	schedule, err := s.store.GetSchedule(scheduleID)
	if err != nil || schedule.CreatedBy != user.ID {
		return nil, &api.APIError{Code: http.StatusForbidden, Message: "forbidden"}
	}

	var request packets.AssignScheduleRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		return nil, &api.APIError{Code: http.StatusBadRequest, Message: err.Error()}
	}

	screen, err := s.store.GetScreenByID(request.ScreenID)
	if err != nil {
		return nil, &api.APIError{Code: http.StatusNotFound, Message: "screen not found"}
	}
	if screen.CreatedBy != user.ID {
		return nil, &api.APIError{Code: http.StatusForbidden, Message: "forbidden"}
	}

	if err := s.store.AssignScheduleToScreen(scheduleID, request.ScreenID); err != nil {
		return nil, &api.APIError{Code: http.StatusInternalServerError, Message: "could not assign schedule to screen"}
	}

	response := gin.H{"message": "assigned"}
	return response, nil
}

func (s *ScheduleController) unassignScheduleFromScreen(ctx *gin.Context, user *model.User) (any, *api.APIError) {
	scheduleID, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		return nil, &api.APIError{Code: http.StatusBadRequest, Message: "invalid schedule id"}
	}

	schedule, err := s.store.GetSchedule(scheduleID)
	if err != nil || schedule.CreatedBy != user.ID {
		return nil, &api.APIError{Code: http.StatusForbidden, Message: "forbidden"}
	}

	screenID, err := strconv.Atoi(ctx.Param("screen_id"))
	if err != nil {
		return nil, &api.APIError{Code: http.StatusBadRequest, Message: "invalid screen id"}
	}

	screen, err := s.store.GetScreenByID(screenID)
	if err != nil {
		return nil, &api.APIError{Code: http.StatusNotFound, Message: "screen not found"}
	}
	if screen.CreatedBy != user.ID {
		return nil, &api.APIError{Code: http.StatusForbidden, Message: "forbidden"}
	}

	if err := s.store.UnassignScheduleFromScreen(scheduleID, screenID); err != nil {
		return nil, &api.APIError{Code: http.StatusInternalServerError, Message: "could not unassign"}
	}

	response := gin.H{"message": "unassigned"}
	return response, nil
}

func (s *ScheduleController) createWindow(ctx *gin.Context, user *model.User) (any, *api.APIError) {
	scheduleID, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		return nil, &api.APIError{Code: http.StatusBadRequest, Message: "invalid schedule id"}
	}

	schedule, err := s.store.GetSchedule(scheduleID)
	if err != nil {
		return nil, &api.APIError{Code: http.StatusNotFound, Message: "schedule not found"}
	}
	if schedule.CreatedBy != user.ID {
		return nil, &api.APIError{Code: http.StatusForbidden, Message: "forbidden"}
	}

	var request packets.CreateWindowRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		return nil, &api.APIError{Code: http.StatusBadRequest, Message: err.Error()}
	}
	if !request.End.After(request.Start) {
		return nil, &api.APIError{Code: http.StatusBadRequest, Message: "end must be after start"}
	}

	playlist, err := s.store.GetPlaylistByID(request.PlaylistID)
	if err != nil {
		return nil, &api.APIError{Code: http.StatusNotFound, Message: "playlist not found"}
	}
	if playlist.CreatedBy != user.ID {
		return nil, &api.APIError{Code: http.StatusForbidden, Message: "forbidden"}
	}

	window, err := s.store.CreateScheduleWindow(
		scheduleID, request.PlaylistID, request.Start, request.End, request.Recurrence, request.RecurUntil, request.Priority,
	)
	if err != nil {
		// 409 with the detailed overlap message (e.g. "overlaps with window 42")
		return nil, &api.APIError{Code: http.StatusConflict, Message: err.Error()}
	}

	return window, nil
}

func (s *ScheduleController) deleteWindow(ctx *gin.Context, user *model.User) (any, *api.APIError) {
	windowID, err := strconv.Atoi(ctx.Param("window_id"))
	if err != nil {
		return nil, &api.APIError{Code: http.StatusBadRequest, Message: "invalid window id"}
	}

	var request packets.DeleteWindowRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		return nil, &api.APIError{Code: http.StatusBadRequest, Message: err.Error()}
	}

	// verify ownership via window -> schedule
	ownedSchedule, err := s.store.GetScheduleByWindowID(windowID)
	if err != nil {
		log.Error().Err(err).Int("window_id", windowID).Msg("deleteWindow ownership check failed")
		return nil, &api.APIError{Code: http.StatusNotFound, Message: "window not found"}
	}
	if ownedSchedule.CreatedBy != user.ID {
		return nil, &api.APIError{Code: http.StatusForbidden, Message: "forbidden"}
	}

	switch request.Scope {
	case "all":
		if err := s.store.DeleteScheduleWindowAll(windowID); err != nil {
			return nil, &api.APIError{Code: http.StatusInternalServerError, Message: "could not delete window series"}
		}
	case "one":
		if request.OccurStart == nil {
			return nil, &api.APIError{Code: http.StatusBadRequest, Message: "occur_start required for scope=one"}
		}
		if err := s.store.DeleteScheduleWindowOneOccurrence(windowID, *request.OccurStart); err != nil {
		 return nil, &api.APIError{Code: http.StatusInternalServerError, Message: "could not delete occurrence"}
		}
	}

	response := gin.H{"message": "deleted"}
	return response, nil
}

func (s *ScheduleController) listOccurrences(ctx *gin.Context, user *model.User) (any, *api.APIError) {
	scheduleID, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		return nil, &api.APIError{Code: http.StatusBadRequest, Message: "invalid schedule id"}
	}

	schedule, err := s.store.GetSchedule(scheduleID)
	if err != nil {
		return nil, &api.APIError{Code: http.StatusNotFound, Message: "schedule not found"}
	}
	if schedule.CreatedBy != user.ID {
		return nil, &api.APIError{Code: http.StatusForbidden, Message: "forbidden"}
	}

	var request packets.ListOccurrencesQuery
	if err := ctx.ShouldBindQuery(&request); err != nil {
		return nil, &api.APIError{Code: http.StatusBadRequest, Message: err.Error()}
	}
	if !request.To.After(request.From) {
		return nil, &api.APIError{Code: http.StatusBadRequest, Message: "to must be after from"}
	}

	occurrences, err := s.store.ListScheduleOccurrences(scheduleID, request.From, request.To)
	if err != nil {
		return nil, &api.APIError{Code: http.StatusInternalServerError, Message: "failed to list occurrences"}
	}

	return occurrences, nil
}

