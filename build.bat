@echo off
echo ========================================================
echo   AWS Kill Switch - Installer & Compiler
echo ========================================================
echo.
echo Compiling aws-kill for Windows...
go build -o aws-kill.exe
if %ERRORLEVEL% NEQ 0 (
    echo.
    echo [ERROR] Compilation failed. Please ensure Go is installed and configured.
    exit /b %ERRORLEVEL%
)
echo.
echo [SUCCESS] aws-kill.exe has been compiled successfully!
echo.
echo To run it from anywhere on your system:
echo 1. Move aws-kill.exe to a folder of your choice (e.g. C:\bin)
echo 2. Add that folder to your System PATH:
echo    - Open Start Search, type "env", and select "Edit the system environment variables"
echo    - Click "Environment Variables"
echo    - Under "System Variables", select "Path" and click "Edit"
echo    - Click "New" and add your folder path (e.g. C:\bin)
echo.
pause
