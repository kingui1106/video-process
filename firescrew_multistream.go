package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/8ff/firescrew/pkg/streamManager"
)

func main() {
	configPath := flag.String("config", "config.json", "Path to configuration file")
	flag.Parse()

	// Create stream manager
	sm, err := streamManager.NewStreamManager(*configPath)
	if err != nil {
		log.Fatalf("Failed to create stream manager: %v", err)
	}

	// Start all enabled cameras
	cameras := sm.GetAllCameras()
	for _, camera := range cameras {
		if camera.Enabled {
			if err := sm.StartStream(camera.ID); err != nil {
				log.Printf("Failed to start stream for camera %s: %v", camera.ID, err)
			}
		}
	}

	// Setup HTTP routes
	mux := http.NewServeMux()
	sm.SetupRoutes(mux)

	// Get web port from config
	port := ":8080"
	if sm.GetConfig().WebPort != "" {
		port = sm.GetConfig().WebPort
	}

	log.Printf("Starting multi-stream server on %s", port)
	log.Printf("Configuration page: http://localhost%s/config", port)
	log.Printf("Stream URLs:")
	for _, camera := range cameras {
		if camera.Enabled {
			log.Printf("  - %s: http://localhost%s/stream/%s", camera.Name, port, camera.ID)
		}
	}

	if err := http.ListenAndServe(port, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
