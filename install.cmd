@echo off
:: ln.bot CLI installer for Windows (CMD)
:: Usage: curl -fsSL https://ln.bot/install.cmd -o install.cmd && install.cmd && del install.cmd

setlocal enabledelayedexpansion

set "REPO=lnbotdev/cli"
set "BINARY=lnbot.exe"
set "INSTALL_DIR=%LOCALAPPDATA%\lnbot"

:: Detect architecture
if "%PROCESSOR_ARCHITECTURE%"=="AMD64" (
    set "ARCH=amd64"
) else if "%PROCESSOR_ARCHITECTURE%"=="ARM64" (
    set "ARCH=arm64"
) else (
    echo Error: unsupported architecture %PROCESSOR_ARCHITECTURE%
    exit /b 1
)

echo Detecting platform... windows/%ARCH%

:: Get latest release tag
for /f "tokens=*" %%i in ('curl -fsSL "https://api.github.com/repos/%REPO%/releases/latest" ^| findstr "tag_name"') do set "TAG_LINE=%%i"
for /f "tokens=2 delims=:," %%a in ("%TAG_LINE%") do set "TAG=%%~a"
set "TAG=%TAG: =%"
set "VERSION=%TAG:v=%"

if "%TAG%"=="" (
    echo Error: could not determine latest release
    exit /b 1
)

echo Latest version: %TAG%

:: Download
set "ARCHIVE=lnbot_windows_%ARCH%.zip"
set "URL=https://github.com/%REPO%/releases/download/%TAG%/%ARCHIVE%"
set "TMPDIR=%TEMP%\lnbot-install-%RANDOM%"

mkdir "%TMPDIR%" 2>nul

echo Downloading %URL%...
curl -fsSL -o "%TMPDIR%\%ARCHIVE%" "%URL%"
if errorlevel 1 (
    echo Error: download failed
    exit /b 1
)

:: Extract
echo Extracting...
powershell -Command "Expand-Archive -Path '%TMPDIR%\%ARCHIVE%' -DestinationPath '%TMPDIR%' -Force"

:: Install
if not exist "%INSTALL_DIR%" mkdir "%INSTALL_DIR%"
move /y "%TMPDIR%\%BINARY%" "%INSTALL_DIR%\%BINARY%" >nul

:: Add to PATH
echo %PATH% | findstr /i /c:"%INSTALL_DIR%" >nul
if errorlevel 1 (
    setx PATH "%PATH%;%INSTALL_DIR%" >nul 2>&1
    set "PATH=%PATH%;%INSTALL_DIR%"
    echo Added %INSTALL_DIR% to PATH
)

:: Cleanup
rmdir /s /q "%TMPDIR%" 2>nul

echo.
echo lnbot %VERSION% installed to %INSTALL_DIR%\%BINARY%
echo.
echo   Get started:
echo     lnbot init
echo     lnbot wallet create --name agent01
echo.
echo   Restart your terminal for PATH changes to take effect.

endlocal
