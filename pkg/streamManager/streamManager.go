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
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// Point represents a 2D point
type Point struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// DrawElement represents a drawable element on the video stream
type DrawElement struct {
	Type      string  `json:"type"`      // "rectangle", "polyline", "text"
	Points    []Point `json:"points"`    // For rectangle: [topLeft, bottomRight], for polyline: multiple points
	Text      string  `json:"text"`      // For text type
	Color     string  `json:"color"`     // Hex color, e.g., "#FF0000"
	Thickness int     `json:"thickness"` // Line thickness in pixels
	FontSize  int     `json:"fontSize"`  // Font size for text
}

// ROI represents a Region of Interest (deprecated, kept for backward compatibility)
type ROI struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

// Camera represents a single camera configuration
type Camera struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	RtspUrl      string        `json:"rtspUrl"`
	ROI          []ROI         `json:"roi"`          // Deprecated, kept for backward compatibility
	DrawElements []DrawElement `json:"drawElements"` // New drawing system
	Enabled      bool          `json:"enabled"`
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

// UpdateCameraROI updates the ROI for a camera (deprecated, use UpdateCameraDrawElements)
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

// UpdateCameraDrawElements updates the DrawElements for a camera
func (sm *StreamManager) UpdateCameraDrawElements(id string, drawElements []DrawElement) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for i := range sm.config.Cameras {
		if sm.config.Cameras[i].ID == id {
			sm.config.Cameras[i].DrawElements = drawElements
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

			// Draw ROI rectangles if configured (backward compatibility)
			if len(camera.ROI) > 0 {
				sm.drawROI(rgba, camera.ROI)
			}

			// Draw new drawing elements
			if len(camera.DrawElements) > 0 {
				sm.drawElements(rgba, camera.DrawElements)
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

// drawElements draws all drawing elements on the image
func (sm *StreamManager) drawElements(img *image.RGBA, elements []DrawElement) {
	for _, elem := range elements {
		switch elem.Type {
		case "rectangle":
			sm.drawRectangleElement(img, elem)
		case "polyline":
			sm.drawPolylineElement(img, elem)
		case "text":
			sm.drawTextElement(img, elem)
		}
	}
}

// parseColor converts hex color string to color.RGBA
func parseColor(hexColor string) color.RGBA {
	// Default to red if parsing fails
	defaultColor := color.RGBA{255, 0, 0, 255}

	if len(hexColor) == 0 {
		return defaultColor
	}

	// Remove # if present
	if hexColor[0] == '#' {
		hexColor = hexColor[1:]
	}

	if len(hexColor) != 6 {
		return defaultColor
	}

	// Parse RGB values
	var r, g, b uint8
	if _, err := fmt.Sscanf(hexColor, "%02x%02x%02x", &r, &g, &b); err != nil {
		return defaultColor
	}

	return color.RGBA{r, g, b, 255}
}

// drawRectangleElement draws a rectangle element
func (sm *StreamManager) drawRectangleElement(img *image.RGBA, elem DrawElement) {
	if len(elem.Points) < 2 {
		return
	}

	x1, y1 := elem.Points[0].X, elem.Points[0].Y
	x2, y2 := elem.Points[1].X, elem.Points[1].Y

	// Ensure x1 < x2 and y1 < y2
	if x1 > x2 {
		x1, x2 = x2, x1
	}
	if y1 > y2 {
		y1, y2 = y2, y1
	}

	col := parseColor(elem.Color)
	thickness := elem.Thickness
	if thickness <= 0 {
		thickness = 2
	}

	colorUniform := image.NewUniform(col)

	// Draw top line
	draw.Draw(img, image.Rect(x1, y1, x2, y1+thickness), colorUniform, image.Point{}, draw.Over)
	// Draw bottom line
	draw.Draw(img, image.Rect(x1, y2-thickness, x2, y2), colorUniform, image.Point{}, draw.Over)
	// Draw left line
	draw.Draw(img, image.Rect(x1, y1, x1+thickness, y2), colorUniform, image.Point{}, draw.Over)
	// Draw right line
	draw.Draw(img, image.Rect(x2-thickness, y1, x2, y2), colorUniform, image.Point{}, draw.Over)
}

// drawPolylineElement draws a polyline element
func (sm *StreamManager) drawPolylineElement(img *image.RGBA, elem DrawElement) {
	if len(elem.Points) < 2 {
		return
	}

	col := parseColor(elem.Color)
	thickness := elem.Thickness
	if thickness <= 0 {
		thickness = 2
	}

	// Draw lines between consecutive points
	for i := 0; i < len(elem.Points)-1; i++ {
		sm.drawLine(img, elem.Points[i].X, elem.Points[i].Y,
			elem.Points[i+1].X, elem.Points[i+1].Y, col, thickness)
	}
}

// drawLine draws a line between two points using Bresenham's algorithm
func (sm *StreamManager) drawLine(img *image.RGBA, x0, y0, x1, y1 int, col color.RGBA, thickness int) {
	dx := abs(x1 - x0)
	dy := abs(y1 - y0)
	sx := -1
	if x0 < x1 {
		sx = 1
	}
	sy := -1
	if y0 < y1 {
		sy = 1
	}
	err := dx - dy

	for {
		// Draw a thick point
		for tx := -thickness / 2; tx <= thickness/2; tx++ {
			for ty := -thickness / 2; ty <= thickness/2; ty++ {
				px, py := x0+tx, y0+ty
				if px >= 0 && px < img.Bounds().Dx() && py >= 0 && py < img.Bounds().Dy() {
					img.Set(px, py, col)
				}
			}
		}

		if x0 == x1 && y0 == y1 {
			break
		}

		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x0 += sx
		}
		if e2 < dx {
			err += dx
			y0 += sy
		}
	}
}

// abs returns the absolute value of an integer
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// drawTextElement draws a text element
func (sm *StreamManager) drawTextElement(img *image.RGBA, elem DrawElement) {
	if len(elem.Points) < 1 || elem.Text == "" {
		return
	}

	x, y := elem.Points[0].X, elem.Points[0].Y
	col := parseColor(elem.Color)

	// Get font size, default to 13
	fontSize := elem.FontSize
	if fontSize <= 0 {
		fontSize = 13
	}

	// Use basicfont for simple text rendering
	// basicfont.Face7x13 is 7 pixels wide and 13 pixels tall
	// We'll scale it by drawing each character as a scaled rectangle
	point := fixed.Point26_6{
		X: fixed.Int26_6(x * 64),
		Y: fixed.Int26_6(y * 64),
	}

	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(col),
		Face: basicfont.Face7x13,
		Dot:  point,
	}

	// If fontSize is different from 13, we need to scale
	// For simplicity, we'll just use the basic font and adjust spacing
	// A better approach would be to load TTF fonts
	if fontSize != 13 {
		// Simple scaling: draw text multiple times with offset for thickness
		scale := float64(fontSize) / 13.0
		for dx := 0; dx < int(scale); dx++ {
			for dy := 0; dy < int(scale); dy++ {
				d.Dot = fixed.Point26_6{
					X: fixed.Int26_6((x + dx) * 64),
					Y: fixed.Int26_6((y + dy) * 64),
				}
				d.DrawString(elem.Text)
			}
		}
	} else {
		d.DrawString(elem.Text)
	}
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
