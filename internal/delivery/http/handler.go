package http

import (
	"encoding/json"
	"net/http"

	"be-remote-device/internal/models"
	"be-remote-device/internal/service"
	"be-remote-device/pkg/response"
)

type DeviceHandler struct {
	svc service.DeviceService
}

func NewDeviceHandler(svc service.DeviceService) *DeviceHandler {
	return &DeviceHandler{svc: svc}
}

func (h *DeviceHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, "API is running smoothly", map[string]string{
		"status": "healthy",
	})
}

func (h *DeviceHandler) ListDevices(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromContext(r.Context())
	devices, err := h.svc.GetAllDevices(userID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, "Devices retrieved successfully", devices)
}

func (h *DeviceHandler) GetDevice(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromContext(r.Context())
	id := r.PathValue("id")
	dev, err := h.svc.GetDeviceByID(userID, id)
	if err != nil {
		response.Error(w, http.StatusNotFound, "Device not found")
		return
	}
	response.JSON(w, http.StatusOK, "Device retrieved successfully", dev)
}

func (h *DeviceHandler) CreateDevice(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromContext(r.Context())
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "Unauthorized: user required to create device")
		return
	}

	var req models.CreateDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	dev, err := h.svc.CreateDevice(userID, req)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	response.JSON(w, http.StatusCreated, "Device registered successfully", dev)
}

func (h *DeviceHandler) UpdateDevice(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromContext(r.Context())
	id := r.PathValue("id")
	var req models.UpdateDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	dev, err := h.svc.UpdateDevice(userID, id, req)
	if err != nil {
		response.Error(w, http.StatusNotFound, err.Error())
		return
	}

	response.JSON(w, http.StatusOK, "Device updated successfully", dev)
}

func (h *DeviceHandler) DeleteDevice(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromContext(r.Context())
	id := r.PathValue("id")
	if err := h.svc.DeleteDevice(userID, id); err != nil {
		response.Error(w, http.StatusNotFound, err.Error())
		return
	}

	response.JSON(w, http.StatusOK, "Device deleted successfully", nil)
}

func (h *DeviceHandler) Heartbeat(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.svc.RecordHeartbeat(id); err != nil {
		response.Error(w, http.StatusNotFound, err.Error())
		return
	}

	response.JSON(w, http.StatusOK, "Heartbeat recorded", nil)
}

func (h *DeviceHandler) SendCommand(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromContext(r.Context())
	id := r.PathValue("id")
	var req models.SendCommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	cmd, err := h.svc.SendCommand(userID, id, req)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	response.JSON(w, http.StatusAccepted, "Command queued", cmd)
}

func (h *DeviceHandler) ListCommands(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromContext(r.Context())
	id := r.PathValue("id")
	cmds, err := h.svc.GetDeviceCommands(userID, id)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	response.JSON(w, http.StatusOK, "Commands retrieved", cmds)
}

type CommandResultRequest struct {
	Status models.CommandStatus `json:"status"`
	Result string               `json:"result"`
}

func (h *DeviceHandler) UpdateCommandResult(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req CommandResultRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	cmd, err := h.svc.UpdateCommandExecution(id, req.Status, req.Result)
	if err != nil {
		response.Error(w, http.StatusNotFound, err.Error())
		return
	}

	response.JSON(w, http.StatusOK, "Command status updated", cmd)
}

func (h *DeviceHandler) RecordTelemetry(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var data models.TelemetryData
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid telemetry payload")
		return
	}
	data.DeviceID = id

	if err := h.svc.RecordTelemetry(data); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	response.JSON(w, http.StatusCreated, "Telemetry recorded", nil)
}

func (h *DeviceHandler) GetTelemetry(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromContext(r.Context())
	id := r.PathValue("id")
	data, err := h.svc.GetLatestTelemetry(userID, id)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if data == nil {
		response.Error(w, http.StatusNotFound, "No telemetry found for this device")
		return
	}

	response.JSON(w, http.StatusOK, "Telemetry retrieved", data)
}
