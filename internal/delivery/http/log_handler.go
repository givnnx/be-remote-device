package http

import (
	"net/http"
	"strconv"

	"be-remote-device/internal/models"
	"be-remote-device/internal/service"
	"be-remote-device/pkg/response"
)

type LogHandler struct {
	svc service.ActionLogService
}

func NewLogHandler(svc service.ActionLogService) *LogHandler {
	return &LogHandler{svc: svc}
}

func (h *LogHandler) ListLogs(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))

	filter := models.ActionLogFilter{
		Actor:  q.Get("actor"),
		Action: q.Get("action"),
		Status: q.Get("status"),
		Limit:  limit,
		Offset: offset,
	}

	logs, total, err := h.svc.GetLogs(filter)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	response.JSON(w, http.StatusOK, "Logs retrieved successfully", map[string]interface{}{
		"total": total,
		"limit": filter.Limit,
		"items": logs,
	})
}

func (h *LogHandler) GetLog(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	logEntry, err := h.svc.GetLogByID(id)
	if err != nil {
		response.Error(w, http.StatusNotFound, "Log not found")
		return
	}

	response.JSON(w, http.StatusOK, "Log retrieved successfully", logEntry)
}
