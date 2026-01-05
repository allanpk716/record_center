@echo off
chcp 65001 >nul
echo ========================================
echo Building record_center...
echo ========================================
echo.

cd /d "%~dp0"

go build -o bin/record_center.exe cmd/record_center/main.go

if %ERRORLEVEL% EQU 0 (
    echo.
    echo ========================================
    echo Build successful!
    echo Output: bin\record_center.exe
    echo ========================================
) else (
    echo.
    echo ========================================
    echo Build failed with error code: %ERRORLEVEL%
    echo ========================================
)

echo.
pause
