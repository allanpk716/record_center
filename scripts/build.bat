@echo off
chcp 65001 >nul
echo ========================================
echo Building record_center...
echo ========================================
echo.

cd /d "%~dp0.."

echo Compiling...
go build -o bin/record_center.exe cmd/record_center/main.go

if %ERRORLEVEL% EQU 0 (
    echo.
    echo Creating runtime directories in bin/...
    if not exist "bin\configs" mkdir "bin\configs"
    if not exist "bin\backups" mkdir "bin\backups"
    if not exist "bin\data" mkdir "bin\data"
    if not exist "bin\logs" mkdir "bin\logs"
    if not exist "bin\temp" mkdir "bin\temp"

    echo.
    echo Copying configuration files...
    copy /Y "configs\backup.yaml" "bin\configs\backup.yaml" >nul

    echo.
    echo ========================================
    echo Build successful!
    echo Output: bin\record_center.exe
    echo Config: bin\configs\backup.yaml
    echo Runtime directories created in bin/
    echo ========================================
) else (
    echo.
    echo ========================================
    echo Build failed with error code: %ERRORLEVEL%
    echo ========================================
)

echo.
pause
