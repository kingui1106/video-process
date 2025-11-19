@echo off
echo Checking what is using port 8080...
netstat -ano | findstr :8080
echo.
echo If you see any results above, note the PID (last column)
echo You can kill that process with: taskkill /F /PID [PID_NUMBER]
pause

