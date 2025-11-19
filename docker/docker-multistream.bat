@echo off
REM Firescrew Multistream Docker Management Script for Windows
REM Usage: docker-multistream.bat [command]

setlocal enabledelayedexpansion

set "SCRIPT_DIR=%~dp0"
set "PROJECT_DIR=%SCRIPT_DIR%.."
set "COMPOSE_FILE=%PROJECT_DIR%\docker-compose.yml"
set "CONFIG_FILE=%PROJECT_DIR%\config.json"

REM Check if docker-compose is installed
where docker-compose >nul 2>nul
if %errorlevel% neq 0 (
    echo [ERROR] docker-compose is not installed. Please install Docker Desktop first.
    exit /b 1
)

REM Parse command
set "COMMAND=%~1"
if "%COMMAND%"=="" set "COMMAND=help"

if /i "%COMMAND%"=="build" goto :build
if /i "%COMMAND%"=="start" goto :start
if /i "%COMMAND%"=="stop" goto :stop
if /i "%COMMAND%"=="restart" goto :restart
if /i "%COMMAND%"=="logs" goto :logs
if /i "%COMMAND%"=="status" goto :status
if /i "%COMMAND%"=="clean" goto :clean
if /i "%COMMAND%"=="help" goto :help
goto :help

:build
echo [INFO] Building Docker image...
cd /d "%PROJECT_DIR%"
docker-compose build
if %errorlevel% equ 0 (
    echo [INFO] Build completed successfully!
) else (
    echo [ERROR] Build failed!
    exit /b 1
)
goto :end

:start
echo [INFO] Starting Firescrew Multistream...
call :check_config
cd /d "%PROJECT_DIR%"
docker-compose up -d
if %errorlevel% equ 0 (
    echo [INFO] Service started successfully!
    echo [INFO] Web interface: http://localhost:8081/config
    timeout /t 2 /nobreak >nul
    start http://localhost:8081/config
) else (
    echo [ERROR] Failed to start service!
    exit /b 1
)
goto :end

:stop
echo [INFO] Stopping Firescrew Multistream...
cd /d "%PROJECT_DIR%"
docker-compose down
if %errorlevel% equ 0 (
    echo [INFO] Service stopped successfully!
) else (
    echo [ERROR] Failed to stop service!
    exit /b 1
)
goto :end

:restart
echo [INFO] Restarting Firescrew Multistream...
call :stop
call :start
goto :end

:logs
echo [INFO] Viewing logs (Press Ctrl+C to exit)...
cd /d "%PROJECT_DIR%"
docker-compose logs -f
goto :end

:status
echo [INFO] Service status:
cd /d "%PROJECT_DIR%"
docker-compose ps
goto :end

:clean
echo [WARN] This will remove all containers, images, and volumes.
set /p "confirm=Are you sure? (y/N): "
if /i "!confirm!"=="y" (
    echo [INFO] Cleaning up...
    cd /d "%PROJECT_DIR%"
    docker-compose down -v --rmi all
    echo [INFO] Cleanup completed!
) else (
    echo [INFO] Cleanup cancelled.
)
goto :end

:help
echo Firescrew Multistream Docker Management Script
echo.
echo Usage: %~nx0 [command]
echo.
echo Commands:
echo     build       Build the Docker image
echo     start       Start the service
echo     stop        Stop the service
echo     restart     Restart the service
echo     logs        View service logs (follow mode)
echo     status      Show service status
echo     clean       Remove containers, images, and volumes
echo     help        Show this help message
echo.
echo Examples:
echo     %~nx0 build
echo     %~nx0 start
echo     %~nx0 logs
echo     %~nx0 stop
echo.
goto :end

:check_config
if not exist "%CONFIG_FILE%" (
    echo [WARN] Config file not found: %CONFIG_FILE%
    echo [WARN] Creating a sample config file...
    (
        echo {
        echo   "webPort": ":8081",
        echo   "cameras": [
        echo     {
        echo       "id": "camera1",
        echo       "name": "示例摄像头",
        echo       "rtspUrl": "rtsp://192.168.1.100:554/stream",
        echo       "roi": [],
        echo       "drawElements": [],
        echo       "enabled": true
        echo     }
        echo   ]
        echo }
    ) > "%CONFIG_FILE%"
    echo [INFO] Sample config created. Please edit %CONFIG_FILE% with your camera settings.
)
exit /b 0

:end
endlocal

