@echo off
setlocal enabledelayedexpansion

echo =========================================
echo Multi-Stream Video Management System
echo =========================================
echo.

REM Check FFmpeg
echo [1/5] Checking FFmpeg...
where ffmpeg >nul 2>&1
if %errorlevel% neq 0 (
    echo [ERROR] FFmpeg not installed
    pause
    exit /b 1
)
echo [OK] FFmpeg installed
echo.

REM Build program
echo [2/5] Building program...
go build -o firescrew_multistream.exe ./firescrew_multistream.go
if %errorlevel% neq 0 (
    echo [ERROR] Build failed
    pause
    exit /b 1
)
echo [OK] Build successful
echo.

REM Check config file
echo [3/5] Checking config file...
if not exist "config_multistream.json" (
    echo [ERROR] Config file not found: config_multistream.json
    pause
    exit /b 1
)
echo [OK] Config file exists
echo.

REM Start service
echo [4/5] Starting service...
echo Service will run in a new window...
start "Firescrew MultiStream" firescrew_multistream.exe -config config_multistream.json
echo [OK] Service started
echo.

REM Wait for service to start
echo [5/5] Waiting for service to start...
timeout /t 3 /nobreak >nul
echo.

REM Display access information
echo =========================================
echo Service Started Successfully!
echo =========================================
echo.
echo Web Config Interface:
echo    http://localhost:8080/config
echo.
echo API Endpoints:
echo    Get camera list: http://localhost:8080/api/cameras
echo.
echo Stream URLs:
echo    http://localhost:8080/stream/camera1
echo    http://localhost:8080/stream/camera2
echo.
echo Test Commands:
echo    REM Get camera list
echo    curl http://localhost:8080/api/cameras
echo.
echo    REM Open stream in browser
echo    start http://localhost:8080/stream/camera1
echo.
echo =========================================
echo.
echo Press any key to open web interface...
pause >nul

start http://localhost:8080/config

echo.
echo Note: Service is running in background
echo To stop service, close the "Firescrew MultiStream" window
echo.
pause

