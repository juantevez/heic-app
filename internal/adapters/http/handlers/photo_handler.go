// internal/adapters/http/handlers/photo_handler.go
package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"heic-photo-processor/internal/domain/entities"
	"heic-photo-processor/internal/domain/ports"

	"github.com/gorilla/mux"
)

type PhotoHandler struct {
	photoService ports.PhotoService
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}

func NewPhotoHandler(photoService ports.PhotoService) *PhotoHandler {
	return &PhotoHandler{
		photoService: photoService,
	}
}

// UploadPhoto handles HEIC photo upload
func (h *PhotoHandler) UploadPhoto(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form (32MB max)
	err := r.ParseMultipartForm(32 << 20)
	if err != nil {
		h.sendErrorResponse(w, http.StatusBadRequest, "Failed to parse multipart form", err)
		return
	}

	// Get the file from form
	file, header, err := r.FormFile("photo")
	if err != nil {
		h.sendErrorResponse(w, http.StatusBadRequest, "No photo file provided", err)
		return
	}
	defer file.Close()

	// Read file data
	fileData, err := io.ReadAll(file)
	if err != nil {
		h.sendErrorResponse(w, http.StatusInternalServerError, "Failed to read file data", err)
		return
	}

	// Get optional bike_id
	var bikeID *int64
	if bikeIDStr := r.FormValue("bike_id"); bikeIDStr != "" {
		if id, err := strconv.ParseInt(bikeIDStr, 10, 64); err == nil {
			bikeID = &id
		}
	}

	// Create upload request
	uploadReq := entities.UploadRequest{
		FileName: header.Filename,
		FileData: fileData,
		BikeID:   bikeID,
	}

	// Process the photo
	response, err := h.photoService.ProcessPhoto(r.Context(), uploadReq)
	if err != nil {
		h.sendErrorResponse(w, http.StatusInternalServerError, "Failed to process photo", err)
		return
	}

	h.sendJSONResponse(w, http.StatusCreated, response)
}

// GetPhoto retrieves a photo by ID
func (h *PhotoHandler) GetPhoto(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	photoID := vars["id"]

	if photoID == "" {
		h.sendErrorResponse(w, http.StatusBadRequest, "Photo ID is required", nil)
		return
	}

	photo, err := h.photoService.GetPhoto(r.Context(), photoID)
	if err != nil {
		h.sendErrorResponse(w, http.StatusNotFound, "Photo not found", err)
		return
	}

	h.sendJSONResponse(w, http.StatusOK, photo)
}

// GetPhotosByBike retrieves all photos for a specific bike
func (h *PhotoHandler) GetPhotosByBike(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	bikeIDStr := vars["bike_id"]

	if bikeIDStr == "" {
		h.sendErrorResponse(w, http.StatusBadRequest, "Bike ID is required", nil)
		return
	}

	bikeID, err := strconv.ParseInt(bikeIDStr, 10, 64)
	if err != nil {
		h.sendErrorResponse(w, http.StatusBadRequest, "Invalid bike ID format", err)
		return
	}

	photos, err := h.photoService.GetPhotosByBike(r.Context(), bikeID)
	if err != nil {
		h.sendErrorResponse(w, http.StatusInternalServerError, "Failed to retrieve photos", err)
		return
	}

	h.sendJSONResponse(w, http.StatusOK, map[string]interface{}{
		"bike_id": bikeID,
		"photos":  photos,
		"count":   len(photos),
	})
}

// DownloadPhoto serves the raw photo file data
func (h *PhotoHandler) DownloadPhoto(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	photoID := vars["id"]

	if photoID == "" {
		h.sendErrorResponse(w, http.StatusBadRequest, "Photo ID is required", nil)
		return
	}

	photo, err := h.photoService.GetPhoto(r.Context(), photoID)
	if err != nil {
		h.sendErrorResponse(w, http.StatusNotFound, "Photo not found", err)
		return
	}

	// Set appropriate headers for file download
	w.Header().Set("Content-Type", "image/heic")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", photo.File.FileName))
	w.Header().Set("Content-Length", strconv.Itoa(len(photo.File.FileData)))

	// Write file data
	_, err = w.Write(photo.File.FileData)
	if err != nil {
		http.Error(w, "Failed to write file data", http.StatusInternalServerError)
		return
	}
}

// HealthCheck endpoint
func (h *PhotoHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "healthy",
		"service":   "heic-photo-processor",
		"timestamp": "2025-01-01T00:00:00Z",
	}
	h.sendJSONResponse(w, http.StatusOK, response)
}

// Helper methods
func (h *PhotoHandler) sendJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "Failed to encode JSON response", http.StatusInternalServerError)
	}
}

func (h *PhotoHandler) sendErrorResponse(w http.ResponseWriter, statusCode int, message string, err error) {
	errorMsg := message
	if err != nil {
		errorMsg = fmt.Sprintf("%s: %s", message, err.Error())
	}

	response := ErrorResponse{
		Error:   http.StatusText(statusCode),
		Message: errorMsg,
		Code:    statusCode,
	}

	h.sendJSONResponse(w, statusCode, response)
}
