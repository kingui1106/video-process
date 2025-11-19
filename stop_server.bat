@echo off
echo Stopping all firescrew_multistream processes...
taskkill /F /IM firescrew_multistream.exe 2>nul
if %errorlevel% equ 0 (
    echo Processes stopped successfully
) else (
    echo No running processes found
)
pause

