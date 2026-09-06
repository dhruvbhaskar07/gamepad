@echo off
title DualSense Server Compiler
cd /d "%~dp0"

echo ========================================================
echo   Building DualSense Mobile Studio Server...
echo ========================================================
echo.

where go >nul 2>nul
if %errorlevel% neq 0 (
    echo [ERROR] Go is not installed or not in PATH!
    echo Please install Go from https://go.dev/dl/
    pause
    exit /b 1
)

echo [1/2] Checking Go module dependencies...
go vet ./server
if %errorlevel% neq 0 (
    echo [WARNING] Go vet found potential issues, attempting build anyway...
)

echo [2/2] Compiling DualSenseServer.exe from ./server...
go build -ldflags="-s -w" -o DualSenseServer.exe ./server

if %errorlevel% equ 0 (
    echo.
    echo ========================================================
    echo   BUILD SUCCESSFUL!
    echo   Binary generated: DualSenseServer.exe
    echo ========================================================
    echo.
) else (
    echo.
    echo [ERROR] Build failed! Check errors above.
    pause
    exit /b %errorlevel%
)
