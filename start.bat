@echo off
echo Building...
go build -o firescrew_multistream.exe ./firescrew_multistream.go

if %errorlevel% neq 0 (
    echo Build failed!
    pause
    exit /b 1
)

echo Starting server...
echo.
echo Web Interface: http://localhost:8080/config
echo Stream URL: http://localhost:8080/stream/camera1
echo.

start "Firescrew MultiStream Server" firescrew_multistream.exe -config config_multistream.json

timeout /t 2 /nobreak >nul
start http://localhost:8080/config

echo.
echo Server is running in a separate window
echo Close that window to stop the server
echo.
pause

