# ln.bot CLI installer for Windows (PowerShell)
# Usage: iwr -useb https://ln.bot/install.ps1 | iex

$ErrorActionPreference = "Stop"

$Repo = "lnbotdev/cli"
$Binary = "lnbot.exe"
$InstallDir = "$env:LOCALAPPDATA\lnbot"

# Detect architecture
$Arch = if ([Environment]::Is64BitOperatingSystem) {
    if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") { "arm64" } else { "amd64" }
} else {
    Write-Error "32-bit Windows is not supported"; exit 1
}

Write-Host "Detecting platform... windows/$Arch" -ForegroundColor Green

# Get latest release
$Release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest"
$Tag = $Release.tag_name
$Version = $Tag.TrimStart("v")

Write-Host "Latest version: $Tag" -ForegroundColor Green

# Download
$Archive = "lnbot_windows_$Arch.zip"
$Url = "https://github.com/$Repo/releases/download/$Tag/$Archive"
$TmpDir = New-Item -ItemType Directory -Path (Join-Path $env:TEMP "lnbot-install-$(Get-Random)")

Write-Host "Downloading $Url..." -ForegroundColor Green

try {
    Invoke-WebRequest -Uri $Url -OutFile (Join-Path $TmpDir $Archive) -UseBasicParsing
} catch {
    Write-Error "Download failed. Check https://github.com/$Repo/releases"
    exit 1
}

# Extract
Write-Host "Extracting..." -ForegroundColor Green
Expand-Archive -Path (Join-Path $TmpDir $Archive) -DestinationPath $TmpDir -Force

# Install
if (-not (Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
}
Move-Item -Path (Join-Path $TmpDir $Binary) -Destination (Join-Path $InstallDir $Binary) -Force

# Add to PATH if not already there
$UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($UserPath -notlike "*$InstallDir*") {
    [Environment]::SetEnvironmentVariable("Path", "$UserPath;$InstallDir", "User")
    $env:Path = "$env:Path;$InstallDir"
    Write-Host "Added $InstallDir to PATH" -ForegroundColor Yellow
}

# Cleanup
Remove-Item -Recurse -Force $TmpDir

Write-Host ""
Write-Host "lnbot $Version installed to $InstallDir\$Binary" -ForegroundColor Green
Write-Host ""
Write-Host "  Get started:" -ForegroundColor Green
Write-Host "    lnbot init" -ForegroundColor Green
Write-Host "    lnbot wallet create --name agent01" -ForegroundColor Green
Write-Host ""
Write-Host "  Restart your terminal for PATH changes to take effect." -ForegroundColor Yellow
