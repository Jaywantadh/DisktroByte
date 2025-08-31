@echo off
echo ========================================
echo    DisktroByte P2P File System
echo ========================================
echo.
echo Starting DisktroByte GUI...
echo.
echo The GUI will open in your browser at:
echo http://localhost:8080
echo.
echo If port 8080 is busy, it will try the next available port.
echo.
echo Press Ctrl+C to stop the server.
echo.

go run ./cmd/cli/main.go

pause
