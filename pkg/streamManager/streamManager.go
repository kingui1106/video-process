package streamManager

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/hybridgroup/mjpeg"
)

// ROI represents a Region of Interest
type ROI struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

// Camera represents a single camera configuration
type Camera struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	RtspUrl string `json:"rtspUrl"`
	ROI     []ROI  `json:"roi"`
	Enabled bool   `json:"enabled"`
}

// Config represents the application configuration
type Config struct {
	WebPort string   `json:"webPort"`
	Cameras []Camera `json:"cameras"`
}

// StreamInfo holds stream and viewer information
type StreamInfo struct {
	Stream      *mjpeg.Stream
	ViewerCount int
	LastViewed  time.Time
	StopTimer   *time.Timer
	mu          sync.Mutex
}

// StreamManager manages multiple camera streams
type StreamManager struct {
	config      *Config
	configPath  string   // Path to the config file
	streams     sync.Map // map[string]*StreamInfo
	mu          sync.RWMutex
	idleTimeout time.Duration // Time to wait before stopping stream when no viewers
}

// NewStreamManager creates a new stream manager
func NewStreamManager(configPath string) (*StreamManager, error) {
	config, err := loadConfig(configPath)
	if err != nil {
		return nil, err
	}

	sm := &StreamManager{
		config:      config,
		configPath:  configPath,
		idleTimeout: 30 * time.Second, // Stop stream after 30 seconds of no viewers
	}

	return sm, nil
}

// loadConfig loads configuration from file
func loadConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// SaveConfig saves configuration to file
func (sm *StreamManager) SaveConfig(path string) error {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// If no path specified, use the original config path
	if path == "" {
		path = sm.configPath
	}

	data, err := json.MarshalIndent(sm.config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetCamera returns a camera by ID
func (sm *StreamManager) GetCamera(id string) (*Camera, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	for i := range sm.config.Cameras {
		if sm.config.Cameras[i].ID == id {
			return &sm.config.Cameras[i], nil
		}
	}
	return nil, fmt.Errorf("camera not found: %s", id)
}

// GetAllCameras returns all cameras
func (sm *StreamManager) GetAllCameras() []Camera {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.config.Cameras
}

// GetConfig returns the configuration
func (sm *StreamManager) GetConfig() *Config {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.config
}

// UpdateCameraROI updates the ROI for a camera
func (sm *StreamManager) UpdateCameraROI(id string, roi []ROI) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for i := range sm.config.Cameras {
		if sm.config.Cameras[i].ID == id {
			sm.config.Cameras[i].ROI = roi
			return nil
		}
	}
	return fmt.Errorf("camera not found: %s", id)
}

// AddCamera adds a new camera to the configuration
func (sm *StreamManager) AddCamera(camera Camera) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Check if camera ID already exists
	for _, c := range sm.config.Cameras {
		if c.ID == camera.ID {
			return fmt.Errorf("camera with ID %s already exists", camera.ID)
		}
	}

	sm.config.Cameras = append(sm.config.Cameras, camera)
	return nil
}

// UpdateCamera updates an existing camera
func (sm *StreamManager) UpdateCamera(id string, camera Camera) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for i := range sm.config.Cameras {
		if sm.config.Cameras[i].ID == id {
			// Keep the same ID
			camera.ID = id
			sm.config.Cameras[i] = camera
			return nil
		}
	}
	return fmt.Errorf("camera not found: %s", id)
}

// DeleteCamera removes a camera from the configuration
func (sm *StreamManager) DeleteCamera(id string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for i := range sm.config.Cameras {
		if sm.config.Cameras[i].ID == id {
			// Stop the stream if it's running
			sm.StopStream(id)

			// Remove from slice
			sm.config.Cameras = append(sm.config.Cameras[:i], sm.config.Cameras[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("camera not found: %s", id)
}

// StopStream stops streaming for a camera
func (sm *StreamManager) StopStream(cameraID string) error {
	if streamInfo, ok := sm.streams.LoadAndDelete(cameraID); ok {
		// Cancel any pending stop timer
		info := streamInfo.(*StreamInfo)
		info.mu.Lock()
		if info.StopTimer != nil {
			info.StopTimer.Stop()
		}
		info.mu.Unlock()

		log.Printf("Stopped stream for camera: %s", cameraID)
		return nil
	}
	return fmt.Errorf("stream not found for camera: %s", cameraID)
}

// AddViewer increments the viewer count for a stream
func (sm *StreamManager) AddViewer(cameraID string) error {
	streamInfo, err := sm.GetStreamInfo(cameraID)
	if err != nil {
		return err
	}

	streamInfo.mu.Lock()
	defer streamInfo.mu.Unlock()

	streamInfo.ViewerCount++
	streamInfo.LastViewed = time.Now()

	// Cancel any pending stop timer
	if streamInfo.StopTimer != nil {
		streamInfo.StopTimer.Stop()
		streamInfo.StopTimer = nil
	}

	log.Printf("Viewer added to camera %s, total viewers: %d", cameraID, streamInfo.ViewerCount)
	return nil
}

// RemoveViewer decrements the viewer count and schedules stream stop if no viewers
func (sm *StreamManager) RemoveViewer(cameraID string) error {
	streamInfo, err := sm.GetStreamInfo(cameraID)
	if err != nil {
		return err
	}

	streamInfo.mu.Lock()
	defer streamInfo.mu.Unlock()

	if streamInfo.ViewerCount > 0 {
		streamInfo.ViewerCount--
	}

	log.Printf("Viewer removed from camera %s, remaining viewers: %d", cameraID, streamInfo.ViewerCount)

	// If no viewers left, schedule stream stop
	if streamInfo.ViewerCount == 0 {
		// Cancel any existing timer
		if streamInfo.StopTimer != nil {
			streamInfo.StopTimer.Stop()
		}

		// Schedule stream stop after idle timeout
		streamInfo.StopTimer = time.AfterFunc(sm.idleTimeout, func() {
			log.Printf("No viewers for %v, stopping stream for camera: %s", sm.idleTimeout, cameraID)
			sm.StopStream(cameraID)
		})
	}

	return nil
}

// GetViewerCount returns the current viewer count for a camera
func (sm *StreamManager) GetViewerCount(cameraID string) (int, error) {
	streamInfo, err := sm.GetStreamInfo(cameraID)
	if err != nil {
		return 0, err
	}

	streamInfo.mu.Lock()
	defer streamInfo.mu.Unlock()

	return streamInfo.ViewerCount, nil
}

// GetStream returns the MJPEG stream for a camera
func (sm *StreamManager) GetStream(cameraID string) (*mjpeg.Stream, error) {
	if streamInfo, ok := sm.streams.Load(cameraID); ok {
		return streamInfo.(*StreamInfo).Stream, nil
	}
	return nil, fmt.Errorf("stream not found for camera: %s", cameraID)
}

// GetStreamInfo returns the stream info for a camera
func (sm *StreamManager) GetStreamInfo(cameraID string) (*StreamInfo, error) {
	if streamInfo, ok := sm.streams.Load(cameraID); ok {
		return streamInfo.(*StreamInfo), nil
	}
	return nil, fmt.Errorf("stream not found for camera: %s", cameraID)
}

// StartStream starts streaming for a camera
func (sm *StreamManager) StartStream(cameraID string) error {
	camera, err := sm.GetCamera(cameraID)
	if err != nil {
		return err
	}

	if !camera.Enabled {
		return fmt.Errorf("camera is disabled: %s", cameraID)
	}

	// Check if stream already exists
	if _, ok := sm.streams.Load(cameraID); ok {
		log.Printf("Stream already running for camera: %s", cameraID)
		return nil
	}

	// Create MJPEG stream
	stream := mjpeg.NewStream()
	streamInfo := &StreamInfo{
		Stream:      stream,
		ViewerCount: 0,
		LastViewed:  time.Now(),
	}
	sm.streams.Store(cameraID, streamInfo)

	// Start processing in goroutine
	go sm.processCamera(camera, stream)

	log.Printf("Started stream for camera: %s (%s)", camera.ID, camera.Name)
	return nil
}

// processCamera processes video frames from a camera
func (sm *StreamManager) processCamera(camera *Camera, stream *mjpeg.Stream) {
	frameChannel := make(chan FrameMsg)

	go func() {
		for {
			sm.processRTSPFeed(camera.RtspUrl, frameChannel)
			time.Sleep(5 * time.Second)
			log.Printf("Restarting RTSP feed for camera: %s", camera.ID)
		}
	}()

	for msg := range frameChannel {
		if msg.Error != "" {
			log.Printf("Error from camera %s: %s", camera.ID, msg.Error)
			continue
		}

		if msg.Frame != nil {
			rgba, ok := msg.Frame.(*image.RGBA)
			if !ok {
				rgba = image.NewRGBA(msg.Frame.Bounds())
				draw.Draw(rgba, rgba.Bounds(), msg.Frame, msg.Frame.Bounds().Min, draw.Src)
			}

			// Draw ROI rectangles if configured
			if len(camera.ROI) > 0 {
				sm.drawROI(rgba, camera.ROI)
			}

			// Encode to JPEG and update stream
			var buf bytes.Buffer
			if err := jpeg.Encode(&buf, rgba, &jpeg.Options{Quality: 80}); err == nil {
				stream.UpdateJPEG(buf.Bytes())
			}
		}
	}
}

// FrameMsg represents a frame message
type FrameMsg struct {
	Frame image.Image
	Error string
}

// processRTSPFeed processes RTSP feed using ffmpeg
func (sm *StreamManager) processRTSPFeed(rtspURL string, msgChannel chan<- FrameMsg) {
	cmd := exec.Command(
		"ffmpeg",
		"-rtsp_transport", "tcp",
		"-re",
		"-i", rtspURL,
		"-analyzeduration", "1000000",
		"-probesize", "1000000",
		"-vf", `select=not(mod(n\,5))`,
		"-fps_mode", "vfr",
		"-c:v", "png",
		"-f", "image2pipe",
		"-",
	)

	stderrBuffer := &bytes.Buffer{}
	cmd.Stderr = stderrBuffer

	pipe, err := cmd.StdoutPipe()
	if err != nil {
		msgChannel <- FrameMsg{Error: err.Error()}
		return
	}
	defer pipe.Close()

	err = cmd.Start()
	if err != nil {
		msgChannel <- FrameMsg{Error: err.Error()}
		return
	}

	frameData := bytes.NewBuffer(nil)
	isFrameStarted := false

	buffer := make([]byte, 8192)
	for {
		n, err := pipe.Read(buffer)
		if err == io.EOF {
			break
		} else if err != nil {
			msgChannel <- FrameMsg{Error: err.Error()}
			return
		}

		frameData.Write(buffer[:n])

		if bytes.HasPrefix(frameData.Bytes(), []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}) {
			isFrameStarted = true
		}

		if isFrameStarted && bytes.HasSuffix(frameData.Bytes(), []byte{0x49, 0x45, 0x4E, 0x44, 0xAE, 0x42, 0x60, 0x82}) {
			img, err := png.Decode(bytes.NewReader(frameData.Bytes()))
			if err != nil {
				msgChannel <- FrameMsg{Error: "Failed to decode PNG: " + err.Error()}
			} else {
				msgChannel <- FrameMsg{Frame: img}
			}

			frameData.Reset()
			isFrameStarted = false
		}
	}

	cmd.Wait()
}

// drawROI draws ROI rectangles on the image
func (sm *StreamManager) drawROI(img *image.RGBA, rois []ROI) {
	// Draw rectangles for each ROI
	for _, roi := range rois {
		sm.drawRect(img, roi.X, roi.Y, roi.Width, roi.Height)
	}
}

// drawRect draws a rectangle on the image
func (sm *StreamManager) drawRect(img *image.RGBA, x, y, width, height int) {
	red := image.NewUniform(color.RGBA{255, 0, 0, 255})

	// Draw top line
	draw.Draw(img, image.Rect(x, y, x+width, y+2), red, image.Point{}, draw.Over)
	// Draw bottom line
	draw.Draw(img, image.Rect(x, y+height-2, x+width, y+height), red, image.Point{}, draw.Over)
	// Draw left line
	draw.Draw(img, image.Rect(x, y, x+2, y+height), red, image.Point{}, draw.Over)
	// Draw right line
	draw.Draw(img, image.Rect(x+width-2, y, x+width, y+height), red, image.Point{}, draw.Over)
}

// ServeHTTP handles HTTP requests for streams
func (sm *StreamManager) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Extract camera ID from URL path
	// Expected format: /stream/{camera_id}
	path := r.URL.Path
	if len(path) < 8 {
		http.Error(w, "Invalid stream path", http.StatusBadRequest)
		return
	}

	cameraID := path[8:] // Remove "/stream/" prefix

	stream, err := sm.GetStream(cameraID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	stream.ServeHTTP(w, r)
}
