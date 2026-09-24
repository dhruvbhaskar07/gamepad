@echo off
setlocal
title Building DualSense Mobile APK...

echo ===================================================
echo   DUALSENSE MOBILE STUDIO - 1-CLICK APK BUILDER
echo ===================================================
echo.

powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0android\build_apk.ps1"

if %ERRORLEVEL% EQU 0 (
    echo.
    echo ===================================================
    echo   BUILD SUCCESSFUL!
    echo   APK Generated: %~dp0DualSenseMobile.apk
    echo ===================================================
) else (
    echo.
    echo [ERROR] Build failed. Please check above logs.
)

echo.
pause
