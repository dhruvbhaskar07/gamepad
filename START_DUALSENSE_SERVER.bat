@echo off
title DualSense Mobile Studio Server
color 0b
cd /d "%~dp0"

echo ==========================================================
echo   🎮 DUALSENSE MOBILE STUDIO - SERVER LAUNCHER
echo ==========================================================
echo.

:: 1. Check if an old zombie process is already running and close it
tasklist /fi "imagename eq DualSenseServer.exe" 2>nul | findstr /i "DualSenseServer.exe" >nul
if %errorlevel% equ 0 (
    echo [INFO] Stopping previous running server instance...
    taskkill /f /im DualSenseServer.exe >nul 2>nul
    timeout /t 1 /nobreak >nul
)

:: 2. Check if DualSenseServer.exe binary exists
if not exist "DualSenseServer.exe" (
    echo [WARNING] DualSenseServer.exe not found!
    echo Compiling server from source now...
    call "build.bat"
    if %errorlevel% neq 0 (
        echo [ERROR] Compilation failed!
        pause
        exit /b 1
    )
)

:: 3. Launch the Server directly in this console window
echo [STARTING] Launching DualSense Controller Server...
echo.
"DualSenseServer.exe"

:: 4. If server stops or closes, pause so user can see what happened
echo.
echo ==========================================================
echo   Server has stopped.
echo ==========================================================
pause
