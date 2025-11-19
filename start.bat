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
echo Using config file: config.json
echo Web Interface: http://localhost:8081/config
echo.

start "Firescrew MultiStream Server" firescrew_multistream.exe

timeout /t 2 /nobreak >nul
start http://localhost:8081/config

echo.
echo Server is running in a separate window
echo Close that window to stop the server
echo.
pause

