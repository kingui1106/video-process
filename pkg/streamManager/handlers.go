package streamManager

import (
	"embed"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

//go:embed static/*
var staticFiles embed.FS

// SetupRoutes sets up HTTP routes for the stream manager
func (sm *StreamManager) SetupRoutes(mux *http.ServeMux) {
	// Serve static files
	mux.HandleFunc("/", sm.handleIndex)
	mux.HandleFunc("/config", sm.handleConfig)
	mux.HandleFunc("/monitor", sm.handleMonitor)
	mux.Handle("/static/", http.FileServer(http.FS(staticFiles)))

	// API routes
	mux.HandleFunc("/api/cameras", sm.handleGetCameras)
	mux.HandleFunc("/api/cameras/", sm.handleCameraAPI)
	mux.HandleFunc("/api/status", sm.handleGetStatus)

	// Stream routes
	mux.HandleFunc("/stream/", sm.handleStream)
}

// handleIndex serves the main page
func (sm *StreamManager) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, "/config", http.StatusFound)
}

// handleConfig serves the configuration page
func (sm *StreamManager) handleConfig(w http.ResponseWriter, r *http.Request) {
	data, err := staticFiles.ReadFile("static/config.html")
	if err != nil {
		http.Error(w, "Failed to load config page", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(data)
}

// handleMonitor serves the monitoring page
func (sm *StreamManager) handleMonitor(w http.ResponseWriter, r *http.Request) {
	data, err := staticFiles.ReadFile("static/monitor.html")
	if err != nil {
		http.Error(w, "Failed to load monitor page", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(data)
}

// handleGetCameras returns all cameras or creates a new camera
func (sm *StreamManager) handleGetCameras(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		cameras := sm.GetAllCameras()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cameras)

	case http.MethodPost:
		// Add new camera
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		var camera Camera
		if err := json.Unmarshal(body, &camera); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		if err := sm.AddCamera(camera); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Save config to file
		if err := sm.SaveConfig(""); err != nil {
			log.Printf("Warning: Failed to save config: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"status": "success", "id": camera.ID})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleCameraAPI handles camera-specific API requests
func (sm *StreamManager) handleCameraAPI(w http.ResponseWriter, r *http.Request) {
	// Extract camera ID from path: /api/cameras/{id}/roi or /api/cameras/{id}
	path := strings.TrimPrefix(r.URL.Path, "/api/cameras/")
	parts := strings.Split(path, "/")

	if len(parts) < 1 {
		http.Error(w, "Invalid API path", http.StatusBadRequest)
		return
	}

	cameraID := parts[0]

	// Handle camera update/delete (no action specified)
	if len(parts) == 1 {
		switch r.Method {
		case http.MethodPut:
			sm.handleUpdateCamera(w, r, cameraID)
		case http.MethodDelete:
			sm.handleDeleteCamera(w, r, cameraID)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	// Handle specific actions
	action := parts[1]
	switch action {
	case "roi":
		sm.handleUpdateROI(w, r, cameraID)
	case "start":
		sm.handleStartStream(w, r, cameraID)
	case "stop":
		sm.handleStopStream(w, r, cameraID)
	default:
		http.Error(w, "Unknown action", http.StatusBadRequest)
	}
}

// handleUpdateROI updates ROI for a camera
func (sm *StreamManager) handleUpdateROI(w http.ResponseWriter, r *http.Request, cameraID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var data struct {
		ROI []ROI `json:"roi"`
	}

	if err := json.Unmarshal(body, &data); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if err := sm.UpdateCameraROI(cameraID, data.ROI); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Save config to file
	if err := sm.SaveConfig(""); err != nil {
		log.Printf("Warning: Failed to save config: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// handleUpdateCamera updates a camera configuration
func (sm *StreamManager) handleUpdateCamera(w http.ResponseWriter, r *http.Request, cameraID string) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var camera Camera
	if err := json.Unmarshal(body, &camera); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if err := sm.UpdateCamera(cameraID, camera); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Save config to file
	if err := sm.SaveConfig(""); err != nil {
		log.Printf("Warning: Failed to save config: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// handleDeleteCamera deletes a camera
func (sm *StreamManager) handleDeleteCamera(w http.ResponseWriter, r *http.Request, cameraID string) {
	if err := sm.DeleteCamera(cameraID); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Save config to file
	if err := sm.SaveConfig(""); err != nil {
		log.Printf("Warning: Failed to save config: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// handleStartStream starts a camera stream
func (sm *StreamManager) handleStartStream(w http.ResponseWriter, r *http.Request, cameraID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := sm.StartStream(cameraID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// handleStopStream stops a camera stream
func (sm *StreamManager) handleStopStream(w http.ResponseWriter, r *http.Request, cameraID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := sm.StopStream(cameraID); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// handleStream handles video stream requests
func (sm *StreamManager) handleStream(w http.ResponseWriter, r *http.Request) {
	// Extract camera ID from path: /stream/{camera_id}
	cameraID := strings.TrimPrefix(r.URL.Path, "/stream/")

	if cameraID == "" {
		http.Error(w, "Camera ID required", http.StatusBadRequest)
		return
	}

	// Remove file extension if present (.flv, .m3u8, etc.)
	if idx := strings.LastIndex(cameraID, "."); idx != -1 {
		cameraID = cameraID[:idx]
	}

	stream, err := sm.GetStream(cameraID)
	if err != nil {
		// Try to start the stream if it doesn't exist
		if err := sm.StartStream(cameraID); err != nil {
			http.Error(w, "Stream not available: "+err.Error(), http.StatusNotFound)
			return
		}

		// Get the stream again
		stream, err = sm.GetStream(cameraID)
		if err != nil {
			http.Error(w, "Failed to start stream", http.StatusInternalServerError)
			return
		}
	}

	// Add viewer
	if err := sm.AddViewer(cameraID); err != nil {
		log.Printf("Failed to add viewer: %v", err)
	}

	// Remove viewer when connection closes
	defer func() {
		if err := sm.RemoveViewer(cameraID); err != nil {
			log.Printf("Failed to remove viewer: %v", err)
		}
	}()

	// Serve MJPEG stream
	stream.ServeHTTP(w, r)
}

// CameraStatus represents the status of a camera
type CameraStatus struct {
	Camera
	IsStreaming bool      `json:"isStreaming"`
	ViewerCount int       `json:"viewerCount"`
	LastViewed  time.Time `json:"lastViewed"`
}

// handleGetStatus returns status of all cameras
func (sm *StreamManager) handleGetStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cameras := sm.GetAllCameras()
	statuses := make([]CameraStatus, 0, len(cameras))

	for _, camera := range cameras {
		status := CameraStatus{
			Camera:      camera,
			IsStreaming: false,
			ViewerCount: 0,
		}

		// Check if stream exists
		if streamInfo, err := sm.GetStreamInfo(camera.ID); err == nil {
			streamInfo.mu.Lock()
			status.IsStreaming = true
			status.ViewerCount = streamInfo.ViewerCount
			status.LastViewed = streamInfo.LastViewed
			streamInfo.mu.Unlock()
		}

		statuses = append(statuses, status)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(statuses)
}
