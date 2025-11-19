package main

import (
	"bytes"
	"context"
	"embed"
	"mime/multipart"
	_ "net/http/pprof"
	"runtime"

	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color/palette"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/8ff/firescrew/pkg/firescrewServe"
	"github.com/8ff/tuna"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/goki/freetype"
	"github.com/goki/freetype/truetype"

	"github.com/8ff/prettyTimer"
	"github.com/hybridgroup/mjpeg"
)

var Version string

var gifSliceMutex sync.Mutex
var gifSlice []image.RGBA

//go:embed assets/*
var assetsFs embed.FS

var stream *mjpeg.Stream

type Config struct {
	CameraName          string       `json:"cameraName"`
	PrintDebug          bool         `json:"printDebug"`
	DeviceUrl           string       `json:"deviceUrl"`
	LoStreamParamBypass StreamParams `json:"loStreamParamBypass"`
	HiResDeviceUrl      string       `json:"hiResDeviceUrl"`
	HiStreamParamBypass StreamParams `json:"hiStreamParamBypass"`
	EnableOutputStream  bool         `json:"enableOutputStream"`
	OutputStreamAddr    string       `json:"outputStreamAddr"`
	Video               struct {
		HiResPath     string `json:"hiResPath"`
		RecodeTsToMp4 bool   `json:"recodeTsToMp4"`
		OnlyRemuxMp4  bool   `json:"onlyRemuxMp4"`
	} `json:"video"`
	Events struct {
		Mqtt struct {
			Host  string `json:"host"`
			Port  int    `json:"port"`
			User  string `json:"user"`
			Pass  string `json:"pass"`
			Topic string `json:"topic"`
		}
		Slack struct {
			Url string `json:"url"`
		}
		ScriptPath string `json:"scriptPath"`
		Webhook    string `json:"webhookUrl"`
	} `json:"events"`
	Notifications struct {
		EnablePushoverAlerts bool   `json:"enablePushoverAlerts"`
		PushoverAppToken     string `json:"pushoverAppToken"`
		PushoverUserKey      string `json:"pushoverUserKey"`
	} `json:"notifications"`
}

type StreamParams struct {
	Width  int
	Height int
	FPS    float64
}

// TODO ADD MUTEX LOCK
type RuntimeConfig struct {
	MotionTriggeredLast time.Time `json:"motionTriggredLast"`
	MotionTriggered     bool      `json:"motionTriggered"`
	HiResControlChannel chan RecordMsg
	MotionVideo         VideoMetadata
	MotionMutex         *sync.Mutex
	TextFont            *truetype.Font
	LoResStreamParams   StreamParams
	HiResStreamParams   StreamParams
	CodecName           string
}

type ControlCommand struct {
	StartRecording bool
	Filename       string
}

type VideoMetadata struct {
	ID           string
	MotionStart  time.Time
	MotionEnd    time.Time
	RecodedToMp4 bool
	Snapshots    []string
	VideoFile    string
	CameraName   string
}

type Event struct {
	Type                string    `json:"type"`
	Timestamp           time.Time `json:"timestamp"`
	MotionTriggeredLast time.Time `json:"motionTriggeredLast"`
	ID                  string    `json:"id"`
	MotionStart         time.Time `json:"motionStart"`
	MotionEnd           time.Time `json:"motionEnd"`
	RecodedToMp4        bool      `json:"recodedToMp4"`
	Snapshots           []string  `json:"snapshots"`
	VideoFile           string    `json:"videoFile"`
	CameraName          string    `json:"cameraName"`
	MetadataPath        string    `json:"metadataPath"`
}

var globalConfig Config
var runtimeConfig RuntimeConfig

type Frame struct {
	Data [][]byte
	Pts  time.Duration
}

type FrameMsg struct {
	Frame image.Image
	Error string
	// Exited   bool
	ExitCode int
}

type StreamInfo struct {
	Streams []struct {
		Width      int     `json:"width"`
		Height     int     `json:"height"`
		CodecType  string  `json:"codec_type"`
		CodecName  string  `json:"codec_name"`
		RFrameRate float64 `json:"-"`
	} `json:"streams"`
}

// RecordMsg struct to control recording
type RecordMsg struct {
	Record   bool
	Filename string
}

func readConfig(path string) Config {
	// Read the configuration file.
	configFile, err := os.ReadFile(path)
	if err != nil {
		Log("error", fmt.Sprintf("Error reading config file: %v", err))
		os.Exit(1)
	}

	// Parse the configuration file into a Config struct.
	var config Config
	err = json.Unmarshal(configFile, &config)
	if err != nil {
		Log("error", fmt.Sprintf("Error parsing config file: %v", err))
		os.Exit(1)
	}

	// Print the configuration properties.
	Log("info", "******************** CONFIG ********************")
	Log("info", fmt.Sprintf("Print Debug: %t", config.PrintDebug))
	Log("info", fmt.Sprintf("Device URL: %s", config.DeviceUrl))
	Log("info", fmt.Sprintf("Lo-Res Param Bypass: Res: %dx%d FPS: %.2f", config.LoStreamParamBypass.Width, config.LoStreamParamBypass.Height, config.LoStreamParamBypass.FPS))
	Log("info", fmt.Sprintf("Hi-Res Param Bypass: Res: %dx%d FPS: %.2f", config.HiStreamParamBypass.Width, config.HiStreamParamBypass.Height, config.HiStreamParamBypass.FPS))
	Log("info", fmt.Sprintf("Hi-Res Device URL: %s", config.HiResDeviceUrl))
	Log("info", fmt.Sprintf("Video HiResPath: %s", config.Video.HiResPath))
	Log("info", fmt.Sprintf("Video RecodeTsToMp4: %t", config.Video.RecodeTsToMp4))
	Log("info", fmt.Sprintf("Video OnlyRemuxMp4: %t", config.Video.OnlyRemuxMp4))
	Log("info", fmt.Sprintf("Enable Output Stream: %t", config.EnableOutputStream))
	Log("info", fmt.Sprintf("Output Stream Address: %s", config.OutputStreamAddr))
	Log("info", "************* EVENTS CONFIG *************")
	Log("info", fmt.Sprintf("Events MQTT Host: %s", config.Events.Mqtt.Host))
	Log("info", fmt.Sprintf("Events MQTT Port: %d", config.Events.Mqtt.Port))
	Log("info", fmt.Sprintf("Events MQTT Topic: %s", config.Events.Mqtt.Topic))
	Log("info", fmt.Sprintf("Events Slack URL: %s", config.Events.Slack.Url))
	Log("info", fmt.Sprintf("Events Script Path: %s", config.Events.ScriptPath))
	Log("info", fmt.Sprintf("Events Webhook URL: %s", config.Events.Webhook))
	Log("info", "************************************************")

	// Load font into runtime
	fontBytes, err := assetsFs.ReadFile("assets/fonts/Changes.ttf")
	if err != nil {
		Log("error", fmt.Sprintf("Error reading font file: %v", err))
		os.Exit(1)
	}

	font, err := freetype.ParseFont(fontBytes)
	if err != nil {
		Log("error", fmt.Sprintf("Error parsing font file: %v", err))
		os.Exit(1)
	}

	runtimeConfig.TextFont = font

	// Check if pushover tokens are provided if enabled
	if config.Notifications.EnablePushoverAlerts {
		if config.Notifications.PushoverAppToken == "" {
			Log("error", fmt.Sprintf("Error parsing config file: %v", errors.New("pushoverAppToken must be set")))
			os.Exit(1)
		}

		if config.Notifications.PushoverUserKey == "" {
			Log("error", fmt.Sprintf("Error parsing config file: %v", errors.New("pushoverUserKey must be set")))
			os.Exit(1)
		}
	}

	return config
}

func eventHandler(eventType string, payload []byte) {
	// Log the event type
	// Log("event", fmt.Sprintf("Event: %s", eventType))

	// Webhook URL
	if globalConfig.Events.Webhook != "" {
		resp, err := http.Post(globalConfig.Events.Webhook, "application/json", bytes.NewReader(payload))
		if err != nil {
			Log("error", fmt.Sprintf("Failed to post to webhook: %s", err))
		} else {
			defer resp.Body.Close()
		}
	}

	// Script Path
	if globalConfig.Events.ScriptPath != "" {
		cmd := exec.Command(globalConfig.Events.ScriptPath)
		stdin, err := cmd.StdinPipe()
		if err != nil {
			Log("error", fmt.Sprintf("Failed to get stdin pipe: %s", err))
			return
		}

		go func() {
			defer stdin.Close()
			_, err := stdin.Write(payload)
			if err != nil {
				Log("error", fmt.Sprintf("Failed to write to stdin: %s", err))
			}
		}()

		if err := cmd.Start(); err != nil {
			Log("error", fmt.Sprintf("Failed to start script: %s", err))
		}
	}

	// Send to Slack
	if globalConfig.Events.Slack.Url != "" {
		slackMessage := map[string]interface{}{
			"text": fmt.Sprintf("Event: %s\nPayload: %s", eventType, string(payload)),
		}
		slackPayload, _ := json.Marshal(slackMessage)
		resp, err := http.Post(globalConfig.Events.Slack.Url, "application/json", bytes.NewReader(slackPayload))
		if err != nil {
			Log("error", fmt.Sprintf("Failed to post to Slack: %s", err))
		} else {
			defer resp.Body.Close()
		}
	}

	// Send to MQTT
	if globalConfig.Events.Mqtt.Host != "" && globalConfig.Events.Mqtt.Port != 0 && globalConfig.Events.Mqtt.Topic != "" {
		err := sendToMQTT(globalConfig.Events.Mqtt.Topic, string(payload), globalConfig.Events.Mqtt.Host, globalConfig.Events.Mqtt.Port, globalConfig.Events.Mqtt.User, globalConfig.Events.Mqtt.Pass)
		if err != nil {
			Log("error", fmt.Sprintf("Failed to send to MQTT: %s", err))
		}
	}
}

func Log(level, msg string) {
	switch level {
	case "info":
		fmt.Printf("\x1b[32m%s [INFO] %s\x1b[0m\n", time.Now().Format("15:04:05"), msg)
	case "notice":
		fmt.Printf("\x1b[35m%s [NOTICE] %s\x1b[0m\n", time.Now().Format("15:04:05"), msg)
	case "event":
		fmt.Printf("\x1b[34m%s [EVENT] %s\x1b[0m\n", time.Now().Format("15:04:05"), msg)
	case "error":
		fmt.Printf("\x1b[31m%s [ERROR] %s\x1b[0m\n", time.Now().Format("15:04:05"), msg)
	case "warning":
		fmt.Printf("\x1b[33m%s [WARNING] %s\x1b[0m\n", time.Now().Format("15:04:05"), msg)
	case "debug":
		if globalConfig.PrintDebug {
			fmt.Printf("\x1b[36m%s [DEBUG] %s\x1b[0m\n", time.Now().Format("15:04:05"), msg)
		}
	default:
		fmt.Printf("%s [UNKNOWN] %s\n", time.Now().Format("15:04:05"), msg)
	}
}

func getStreamInfo(rtspURL string) (StreamInfo, error) {
	// Create a context that will time out
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ffprobe", "-rtsp_transport", "tcp", "-v", "quiet", "-print_format", "json", "-show_streams", rtspURL)
	output, err := cmd.Output()
	if err != nil {
		Log("debug", fmt.Sprintf("ffprobe output: %s", output))
		return StreamInfo{}, err
	}

	Log("debug", fmt.Sprintf("ffprobe url: %s output: %s", rtspURL, output))

	// Unmarshal into a temporary structure to get the raw frame rate
	var rawInfo struct {
		Streams []struct {
			Width      int    `json:"width"`
			Height     int    `json:"height"`
			CodecType  string `json:"codec_type"`
			CodecName  string `json:"codec_name"`
			RFrameRate string `json:"r_frame_rate"`
		} `json:"streams"`
	}
	if err := json.Unmarshal(output, &rawInfo); err != nil {
		return StreamInfo{}, err
	}

	// Process the streams, converting the frame rate and filtering as needed
	var info StreamInfo
	for _, stream := range rawInfo.Streams {
		if stream.Width == 0 || stream.Height == 0 {
			continue // Skip streams with zero values
		}
		frParts := strings.Split(stream.RFrameRate, "/")
		if len(frParts) == 2 {
			numerator, err1 := strconv.Atoi(frParts[0])
			denominator, err2 := strconv.Atoi(frParts[1])
			if err1 != nil || err2 != nil || denominator == 0 {
				return StreamInfo{}, fmt.Errorf("invalid frame rate: %s", stream.RFrameRate)
			}
			frameRate := float64(numerator) / float64(denominator) // Calculate FPS
			info.Streams = append(info.Streams, struct {
				Width      int     `json:"width"`
				Height     int     `json:"height"`
				CodecType  string  `json:"codec_type"`
				CodecName  string  `json:"codec_name"`
				RFrameRate float64 `json:"-"`
			}{
				Width:      stream.Width,
				Height:     stream.Height,
				CodecType:  stream.CodecType,
				CodecName:  stream.CodecName,
				RFrameRate: frameRate,
			})
		}
	}

	return info, nil
}

func CheckFFmpegAndFFprobe() (bool, error) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		// Print PATH
		path := os.Getenv("PATH")
		Log("error", fmt.Sprintf("PATH: %s", path))
		return false, fmt.Errorf("ffmpeg binary not found: %w", err)
	}

	if _, err := exec.LookPath("ffprobe"); err != nil {
		// Print PATH
		path := os.Getenv("PATH")
		Log("error", fmt.Sprintf("PATH: %s", path))
		return false, fmt.Errorf("ffprobe binary not found: %w", err)
	}

	return true, nil
}

func processRTSPFeed(rtspURL string, msgChannel chan<- FrameMsg) {
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

	frameCount := 0
	frameData := bytes.NewBuffer(nil)
	isFrameStarted := false

	buffer := make([]byte, 8192) // Buffer size
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

			frameCount++
			frameData.Reset()
			isFrameStarted = false
		}

		if frameData.Len() > 2*1024*1024 {
			startIdx := bytes.Index(frameData.Bytes(), []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A})
			if startIdx != -1 {
				frameData.Next(startIdx)
				isFrameStarted = true
			} else {
				frameData.Reset()
				isFrameStarted = false
			}
		}
	}

	err = cmd.Wait()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				exitCode = status.ExitStatus()
			}
		}
		msgChannel <- FrameMsg{Error: "FFmpeg exited with error: " + err.Error(), ExitCode: exitCode}
	}

	if stderrBuffer.Len() > 0 {
		msgChannel <- FrameMsg{Error: "FFmpeg STDERR: " + stderrBuffer.String()}
	}
}

func recordRTSPStream(rtspURL string, controlChannel <-chan RecordMsg, prebufferDuration time.Duration) {
	var file *os.File
	recording := false

	cmd := exec.Command("ffmpeg", "-rtsp_transport", "tcp", "-i", rtspURL, "-c", "copy", "-f", "mpegts", "pipe:1")
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		Log("error", fmt.Sprintf("Error creating pipe: %v", err))
		return
	}

	err = cmd.Start()
	if err != nil {
		Log("error", fmt.Sprintf("Error starting ffmpeg: %v", err))
		return
	}

	defer func() {
		if recording && file != nil {
			file.Close()
		}
		cmd.Wait()
	}()

	type chunkInfo struct {
		Data []byte
		Time time.Time
	}

	bufferSize := 4096
	prebuffer := make([]chunkInfo, 0)
	buffer := make([]byte, bufferSize)

	for {
		select {
		case msg := <-controlChannel:
			if msg.Record && !recording {
				file, err = os.Create(msg.Filename)
				if err != nil {
					log.Fatal(err)
					return
				}
				for _, chunk := range prebuffer { // Write prebuffered data
					_, err := file.Write(chunk.Data)
					if err != nil {
						log.Fatal(err)
						return
					}
				}
				recording = true
			} else if !msg.Record && recording {
				file.Close()
				recording = false
			}

		default:
			n, err := pipe.Read(buffer)
			if err != nil {
				if err == io.EOF {
					return
				}
				log.Fatal(err)
				return
			}

			// Prebuffer handling
			chunk := make([]byte, n)
			copy(chunk, buffer[:n])
			timestamp := time.Now()
			prebuffer = append(prebuffer, chunkInfo{Data: chunk, Time: timestamp})
			// Remove chunks that are older than prebufferDuration
			for len(prebuffer) > 1 && timestamp.Sub(prebuffer[0].Time) > prebufferDuration {
				prebuffer = prebuffer[1:]
			}

			if recording && file != nil {
				_, err := file.Write(buffer[:n])
				if err != nil {
					log.Fatal(err)
					return
				}
			}
		}
	}
}

func recodeToMP4(inputFile string) (string, error) {
	// Check if the input file has a .ts extension
	if !strings.HasSuffix(inputFile, ".ts") {
		return "", fmt.Errorf("input file must have a .ts extension. Got: %s", inputFile)
	}

	// Remove the .ts extension and replace it with .mp4
	outputFile := strings.TrimSuffix(inputFile, ".ts") + ".mp4"

	var cmd *exec.Cmd
	// Create the FFmpeg command
	if globalConfig.Video.OnlyRemuxMp4 {
		if runtimeConfig.CodecName == "hevc" {
			cmd = exec.Command("ffmpeg", "-i", inputFile,
				"-c:v", "copy",
				"-c:a", "aac",
				"-tag:v", "hvc1",
				"-movflags", "+faststart",
				"-hls_segment_type", "fmp4",
				outputFile)
		} else {
			cmd = exec.Command("ffmpeg", "-i", inputFile, "-c", "copy", outputFile)
		}
	} else {
		cmd = exec.Command("ffmpeg", "-i", inputFile, "-c:v", "libx264", "-c:a", "aac", outputFile)
	}

	// Capture the standard output and standard error
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("FFmpeg command failed: %v\n%s", err, output)
	}

	return outputFile, nil
}

func main() {
	ptime := prettyTimer.NewTimingStats()
	// Check if there is a config file argument, if there isnt give error and exit
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Not enough arguments provided\n")
		fmt.Println("Usage: firescrew [configfile]")
		fmt.Println("  -t, --template, t\tPrints the template config to stdout")
		fmt.Println("  -h, --help, h\t\tPrints this help message")
		fmt.Println("  -s, --serve, s\tStarts the web server, requires: [path] [addr]")
		return
	}

	switch os.Args[1] {
	case "-t", "--template", "t":
		// Dump template config to stdout
		printTemplateFile()
		return
	case "-h", "--help", "h":
		// Print help
		fmt.Println("Usage: firescrew [configfile]")
		fmt.Println("  -t, --template, t\tPrints the template config to stdout")
		fmt.Println("  -h, --help, h\t\tPrints this help message")
		fmt.Println("  -s, --serve, s\tStarts the web server, requires: [path] [addr]")
		fmt.Println("  -v, --version, v\tPrints the version")
		fmt.Println("  -update, --update, update\tUpdates firescrew to the latest version")
		return
	case "-s", "--serve", "s":
		// This requires 2 more params, a path to files and an addr in form :8080
		// Check if those params are provided if not give help message
		if len(os.Args) < 4 {
			fmt.Fprintf(os.Stderr, "Not enough arguments provided\n")
			fmt.Fprintf(os.Stderr, ("Usage: firescrew -s [path] [addr]\n"))
			return
		}
		err := firescrewServe.Serve(os.Args[2], os.Args[3])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error starting server: %v\n", err)
			return
		}
		os.Exit(1)
	case "-v", "--version", "v":
		// Print version
		fmt.Println(Version)
		os.Exit(0)
	case "-update", "--update", "update":
		// Determine OS and ARCH
		osRelease := runtime.GOOS
		arch := runtime.GOARCH

		// Build URL
		e := tuna.SelfUpdate(fmt.Sprintf("https://github.com/8ff/firescrew/releases/download/latest/firescrew.%s.%s", osRelease, arch))
		if e != nil {
			fmt.Println(e)
			os.Exit(1)
		}

		fmt.Println("Updated!")
		os.Exit(0)
	}

	// Read the config file
	globalConfig = readConfig(os.Args[1])

	// Check if ffmpeg/ffprobe binaries are available
	_, err := CheckFFmpegAndFFprobe()
	if err != nil {
		Log("error", fmt.Sprintf("Unable to find ffmpeg/ffprobe binaries. Please install them: %s", err))
		os.Exit(2)
	}

	if globalConfig.LoStreamParamBypass.Width == 0 || globalConfig.LoStreamParamBypass.Height == 0 || globalConfig.LoStreamParamBypass.FPS == 0 {
		// Print HI/LO stream details
		hiResStreamInfo, err := getStreamInfo(globalConfig.HiResDeviceUrl)
		if err != nil {
			Log("error", fmt.Sprintf("Error getting stream info: ffprobe: %v", err))
			os.Exit(3)
		}

		if len(hiResStreamInfo.Streams) == 0 {
			Log("error", fmt.Sprintf("No HI res streams found at %s", globalConfig.HiResDeviceUrl))
			os.Exit(3)
		}

		// Find stream with codec_type: video
		streamIndex := -1
		for index, stream := range hiResStreamInfo.Streams {
			if stream.CodecType == "video" {
				streamIndex = index
				runtimeConfig.CodecName = stream.CodecName
				if globalConfig.Video.OnlyRemuxMp4 {
					if stream.CodecName != "h264" {
						Log("warning", fmt.Sprintf("OnlyRemuxMp4 is enabled but the stream codec is not h264 or h265. Your videos may not play in WebUI. Codec: %s", stream.CodecName))
					}
				}
				break
			}
		}

		if streamIndex == -1 {
			Log("error", fmt.Sprintf("No video stream found at %s", globalConfig.HiResDeviceUrl))
			os.Exit(3)
		}

		runtimeConfig.HiResStreamParams = StreamParams{
			Width:  hiResStreamInfo.Streams[streamIndex].Width,
			Height: hiResStreamInfo.Streams[streamIndex].Height,
			FPS:    hiResStreamInfo.Streams[streamIndex].RFrameRate,
		}
	} else {
		runtimeConfig.HiResStreamParams = globalConfig.HiStreamParamBypass
	}

	if globalConfig.LoStreamParamBypass.Width == 0 || globalConfig.LoStreamParamBypass.Height == 0 || globalConfig.LoStreamParamBypass.FPS == 0 {
		loResStreamInfo, err := getStreamInfo(globalConfig.DeviceUrl)
		if err != nil {
			Log("error", fmt.Sprintf("Error getting stream info: %v", err))
			os.Exit(3)
		}

		if len(loResStreamInfo.Streams) == 0 {
			Log("error", fmt.Sprintf("No LO res streams found at %s", globalConfig.DeviceUrl))
			os.Exit(3)
		}

		// Find stream with codec_type: video
		streamIndex := -1
		for index, stream := range loResStreamInfo.Streams {
			if stream.CodecType == "video" {
				streamIndex = index
				break
			}
		}

		if streamIndex == -1 {
			Log("error", fmt.Sprintf("No video stream found at %s", globalConfig.DeviceUrl))
			os.Exit(3)
		}

		runtimeConfig.LoResStreamParams = StreamParams{
			Width:  loResStreamInfo.Streams[streamIndex].Width,
			Height: loResStreamInfo.Streams[streamIndex].Height,
			FPS:    loResStreamInfo.Streams[streamIndex].RFrameRate,
		}
	} else {
		runtimeConfig.LoResStreamParams = globalConfig.LoStreamParamBypass
	}

	// Print stream info from runtimeConfig
	Log("info", "******************** STREAM INFO ********************")
	Log("info", fmt.Sprintf("Lo-Res Stream Resolution: %dx%d FPS: %.2f", runtimeConfig.LoResStreamParams.Width, runtimeConfig.LoResStreamParams.Height, runtimeConfig.LoResStreamParams.FPS))
	Log("info", fmt.Sprintf("Hi-Res Stream Resolution: %dx%d FPS: %.2f", runtimeConfig.HiResStreamParams.Width, runtimeConfig.HiResStreamParams.Height, runtimeConfig.HiResStreamParams.FPS))
	Log("info", "*****************************************************")

	// Define motion mutex
	runtimeConfig.MotionMutex = &sync.Mutex{}

	stream = mjpeg.NewStream()
	if globalConfig.EnableOutputStream {
		go startWebcamStream(stream)
	}

	// Start HI Res prebuffering
	runtimeConfig.HiResControlChannel = make(chan RecordMsg)
	go func() {
		for {
			recordRTSPStream(globalConfig.HiResDeviceUrl, runtimeConfig.HiResControlChannel, 5*time.Second)
			// defer close(runtimeConfig.HiResControlChannel)
			time.Sleep(5 * time.Second)
			Log("warning", "Restarting HI RTSP feed")
		}
	}()

	frameChannel := make(chan FrameMsg)
	go func(frameChannel chan FrameMsg) {
		for {
			processRTSPFeed(globalConfig.DeviceUrl, frameChannel)
			// Log("warning", "EXITED")
			//*********** EXITS BELOW ***********//
			time.Sleep(5 * time.Second)
			Log("warning", "Restarting LO RTSP feed")
		}
	}(frameChannel)
	// go dumpRtspFrames(globalConfig.DeviceUrl, "/Volumes/RAMDisk/", 4) // 1 means mod every nTh frame
	// go readFramesFromRam(frameChannel, "/Volumes/RAMDisk/")

	for msg := range frameChannel {
		if msg.Error != "" {
			Log("error", msg.Error)
			continue
		}

		if msg.Frame != nil {
			ptime.Start() // DEBUG TIMER

			rgba, ok := msg.Frame.(*image.RGBA)
			if !ok {
				// Convert to RGBA if it's not already
				rgba = image.NewRGBA(msg.Frame.Bounds())
				draw.Draw(rgba, rgba.Bounds(), msg.Frame, msg.Frame.Bounds().Min, draw.Src)
			}

			// Stream the image to the web if enabled
			if globalConfig.EnableOutputStream {
				streamImage(rgba, stream)
			}

			ptime.Finish() // DEBUG TIMER
			// ptime.PrintStats() // DEBUG TIMER

		}
	}

}

func streamImage(img *image.RGBA, stream *mjpeg.Stream) {
	// Encode the RGBA image to JPEG
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		// Handle encoding error
		return
	}

	// Stream video over HTTP
	stream.UpdateJPEG(buf.Bytes())
}

func startWebcamStream(stream *mjpeg.Stream) {
	// start http server
	http.Handle("/", stream)

	server := &http.Server{
		Addr:         globalConfig.OutputStreamAddr,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	log.Fatal(server.ListenAndServe())
}

func printTemplateFile() {
	fileBytes, err := assetsFs.ReadFile("assets/template.json")
	if err != nil {
		log.Fatalf("Failed to read template file: %v", err)
	}

	fmt.Println(string(fileBytes))
}

func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz" + "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	seededRand := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func CountChangedPixels(img1, img2 *image.RGBA, threshold uint8) int {
	if img1.Bounds() != img2.Bounds() {
		return -1
	}

	count := 0
	for y := 0; y < img1.Bounds().Dy(); y++ {
		for x := 0; x < img1.Bounds().Dx(); x++ {
			offset := y*img1.Stride + x*4
			r1, g1, b1 := int(img1.Pix[offset]), int(img1.Pix[offset+1]), int(img1.Pix[offset+2])
			r2, g2, b2 := int(img2.Pix[offset]), int(img2.Pix[offset+1]), int(img2.Pix[offset+2])
			gray1 := (299*r1 + 587*g1 + 114*b1) / 1000
			gray2 := (299*r2 + 587*g2 + 114*b2) / 1000
			diff := gray1 - gray2
			if diff < 0 {
				diff = -diff
			}
			if uint8(diff) > threshold {
				count++
			}
		}
	}

	return count
}

func saveJPEG(filename string, img *image.RGBA, quality int) {
	file, err := os.Create(filename)
	if err != nil {
		Log("error", fmt.Sprintf("File create error: %s", err))
		return
	}
	defer file.Close()

	options := &jpeg.Options{Quality: quality} // Quality ranges from 1 to 100
	err = jpeg.Encode(file, img, options)
	if err != nil {
		Log("error", fmt.Sprintf("JPEG encode error: %s", err))
		return
	}
}

func sendToMQTT(topic string, message string, host string, port int, user string, pass string) error {
	// MQTT client options
	opts := mqtt.NewClientOptions().AddBroker(fmt.Sprintf("tcp://%s:%d", host, port))

	if user != "" && pass != "" {
		opts.SetUsername(user)
		opts.SetPassword(pass)
	}

	// Create and connect the client
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		Log("error", fmt.Sprintf("Failed to connect to MQTT: %s", token.Error()))
		return token.Error() // Return the connection error
	}
	defer client.Disconnect(250)

	// Publish the message
	token := client.Publish(topic, 0, false, message)
	token.Wait()
	if token.Error() != nil {
		Log("error", fmt.Sprintf("Failed to publish to MQTT: %s", token.Error()))
		return token.Error() // Return the publishing error
	}

	return nil // Return nil if there were no errors
}

func sendPushoverNotification(userKey string, appToken string, msg string, img *image.RGBA) error {
	// Convert the image to JPEG format
	var imgBuffer bytes.Buffer
	if err := jpeg.Encode(&imgBuffer, img, nil); err != nil {
		return fmt.Errorf("error encoding image: %v", err)
	}

	// Create a new HTTP request
	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	// Set up form fields
	if err := w.WriteField("token", appToken); err != nil {
		return fmt.Errorf("WriteField Error: %v", err)
	}
	if err := w.WriteField("user", userKey); err != nil {
		return fmt.Errorf("WriteField Error: %v", err)
	}
	if err := w.WriteField("message", msg); err != nil {
		return fmt.Errorf("WriteField Error: %v", err)
	}

	// Attach the in-memory image as an attachment
	fw, err := w.CreateFormFile("attachment", "image.jpg")
	if err != nil {
		return fmt.Errorf("CreateFormFile Error: %v", err)
	}
	if _, err := io.Copy(fw, &imgBuffer); err != nil {
		return fmt.Errorf("copy File Error: %v", err)
	}

	// Close the writer
	w.Close()

	// Create a HTTP request to Pushover API
	req, err := http.NewRequest("POST", "https://api.pushover.net/1/messages.json", &b)
	if err != nil {
		return fmt.Errorf("NewRequest Error: %v", err)
	}

	// Set the content type, this will include the boundary.
	req.Header.Set("Content-Type", w.FormDataContentType())

	// Execute the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("error executing request: %v", err)
	}
	defer resp.Body.Close()

	// Check the response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Non-OK HTTP status: %s", resp.Status)
	}

	return nil
}

// CreateGIF creates a GIF file from a slice of *image.RGBA images
func CreateGIF(images []image.RGBA, outputPath string, delay int) error {
	outFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	anim := &gif.GIF{}
	for _, srcImg := range images {
		// Convert image.RGBA to *image.Paletted
		bounds := srcImg.Bounds()
		palettedImage := image.NewPaletted(bounds, palette.Plan9)
		draw.Draw(palettedImage, palettedImage.Rect, &srcImg, bounds.Min, draw.Over)

		anim.Image = append(anim.Image, palettedImage)
		anim.Delay = append(anim.Delay, delay)
	}

	return gif.EncodeAll(outFile, anim)
}

func sendPushoverNotificationGif(userKey string, appToken string, msg string, gifPath string) error {
	// Open the GIF file
	file, err := os.Open(gifPath)
	if err != nil {
		return fmt.Errorf("Error opening GIF file: %v", err)
	}
	defer file.Close()

	// Create a new HTTP request
	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	// Set up form fields
	if err := w.WriteField("token", appToken); err != nil {
		return fmt.Errorf("WriteField Error: %v", err)
	}
	if err := w.WriteField("user", userKey); err != nil {
		return fmt.Errorf("WriteField Error: %v", err)
	}
	if err := w.WriteField("message", msg); err != nil {
		return fmt.Errorf("WriteField Error: %v", err)
	}

	// Attach the GIF file as an attachment
	fw, err := w.CreateFormFile("attachment", "image.gif")
	if err != nil {
		return fmt.Errorf("CreateFormFile Error: %v", err)
	}
	if _, err := io.Copy(fw, file); err != nil {
		return fmt.Errorf("Copy File Error: %v", err)
	}

	// Close the writer
	w.Close()

	// Create a HTTP request to Pushover API
	req, err := http.NewRequest("POST", "https://api.pushover.net/1/messages.json", &b)
	if err != nil {
		return fmt.Errorf("NewRequest Error: %v", err)
	}

	// Set the content type, this will include the boundary.
	req.Header.Set("Content-Type", w.FormDataContentType())

	// Execute the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("Error executing request: %v", err)
	}
	defer resp.Body.Close()

	// Check the response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Non-OK HTTP status: %s", resp.Status)
	}

	return nil
}
