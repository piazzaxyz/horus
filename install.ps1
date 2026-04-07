# install.ps1 - Install QAITOR on Windows (PowerShell)
#Requires -Version 5.0

$BinaryName = "qaitor.exe"
$BuildDir = "bin"
$InstallDir = Join-Path $env:USERPROFILE "bin"

Write-Host "QAITOR Installer" -ForegroundColor Cyan
Write-Host "================" -ForegroundColor Cyan

# Check for Go
try {
    $GoVersion = (go version) 2>&1
    Write-Host "Go: $GoVersion" -ForegroundColor Green
} catch {
    Write-Host "ERROR: Go is not installed. Please install Go 1.22+ from https://go.dev/dl/" -ForegroundColor Red
    exit 1
}

# Build
Write-Host "Building QAITOR..." -ForegroundColor Yellow
New-Item -ItemType Directory -Force -Path $BuildDir | Out-Null
& go build -ldflags "-s -w" -o "$BuildDir\$BinaryName" .

if ($LASTEXITCODE -ne 0) {
    Write-Host "ERROR: Build failed." -ForegroundColor Red
    exit 1
}

Write-Host "Build successful: $BuildDir\$BinaryName" -ForegroundColor Green

# Create install dir
New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null

# Copy binary
Copy-Item -Path "$BuildDir\$BinaryName" -Destination "$InstallDir\$BinaryName" -Force
Write-Host "Installed to: $InstallDir\$BinaryName" -ForegroundColor Green

# Check PATH
$CurrentPath = [Environment]::GetEnvironmentVariable("PATH", "User")
if ($CurrentPath -notlike "*$InstallDir*") {
    Write-Host ""
    Write-Host "Adding $InstallDir to your user PATH..." -ForegroundColor Yellow
    $NewPath = "$CurrentPath;$InstallDir"
    [Environment]::SetEnvironmentVariable("PATH", $NewPath, "User")
    Write-Host "Done. Restart PowerShell for PATH changes to take effect." -ForegroundColor Green
} else {
    Write-Host "PATH already contains $InstallDir" -ForegroundColor Green
}

Write-Host ""
Write-Host "Installation complete! Run 'qaitor' to start QAITOR." -ForegroundColor Cyan
