@echo off
echo SoundCloud MP3 Downloader
echo ========================
echo.

if "%~1"=="" (
    echo Usage: download.bat "SoundCloud_URL"
    echo Example: download.bat "https://soundcloud.com/artist/track-name"
    pause
    exit /b 1
)

echo Downloading: %1
echo.

soundcloud-downloader.exe %1

if %ERRORLEVEL% EQU 0 (
    echo.
    echo Download completed successfully!
) else (
    echo.
    echo Download failed. Please check the error message above.
)

pause 