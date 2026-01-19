# --- nanocode-go Windows PowerShell Installer Script ---

# This script must be run with Administrator privileges for the PATH and setx commands to work correctly.

# Configuration
$EXEC_NAME = "nanocode.exe"
$TOOLS_DIR = "C:\Tools"

Write-Host "--- nanocode-go Windows PowerShell Installer ---" -ForegroundColor Cyan

# --- 1. Ask for API Key ---
$API_KEY = Read-Host "Enter your Mistral API Key"

if ([string]::IsNullOrEmpty($API_KEY)) {
    Write-Host "Error: API Key cannot be empty. Exiting." -ForegroundColor Red
    exit 1
}

# --- 2. Check for Go installation ---
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Host "Go is not installed. Installing Go..." -ForegroundColor Yellow
    # Download and install Go
    $goInstallerUrl = "https://go.dev/dl/go1.21.6.windows-amd64.msi"
    $goInstallerPath = "$env:TEMP\go1.21.6.windows-amd64.msi"
    Invoke-WebRequest -Uri $goInstallerUrl -OutFile $goInstallerPath
    Start-Process -FilePath "msiexec.exe" -ArgumentList "/i $goInstallerPath /quiet" -Wait
    Write-Host "Go has been installed." -ForegroundColor Green
}
Write-Host "Go installation found." -ForegroundColor Green

# --- 3. Build the project ---
Write-Host "Building nanocode executable..." -ForegroundColor Yellow
go build -o $EXEC_NAME nanocode.go
if ($LASTEXITCODE -ne 0) {
    Write-Host "Error: Go build failed. Check nanocode.go for errors." -ForegroundColor Red
    exit 1
}
Write-Host "Build successful." -ForegroundColor Green

# --- 4. Create tools folder and move the file ---
if (-not (Test-Path $TOOLS_DIR)) {
    Write-Host "Creating tools directory $TOOLS_DIR" -ForegroundColor Yellow
    New-Item -Path $TOOLS_DIR -ItemType Directory | Out-Null
}

Write-Host "Moving $EXEC_NAME to $TOOLS_DIR" -ForegroundColor Yellow
Move-Item -Path $EXEC_NAME -Destination $TOOLS_DIR -Force
if ($LASTEXITCODE -ne 0) {
    Write-Host "Error: Failed to move executable." -ForegroundColor Red
    exit 1
}

# --- 5. Add that folder to your System PATH (User Scope) ---
Write-Host "Adding $TOOLS_DIR to your User PATH (requires elevated privileges)..." -ForegroundColor Yellow
$currentPath = [Environment]::GetEnvironmentVariable("Path", "User")
if (-not ($currentPath -like "*$TOOLS_DIR*")) {
    [Environment]::SetEnvironmentVariable("Path", "$currentPath;$TOOLS_DIR", "User")
    Write-Host "PATH updated." -ForegroundColor Green
} else {
    Write-Host "PATH already contains $TOOLS_DIR. Skipping update." -ForegroundColor Green
}

# --- 6. Set the API Key permanently (User Scope) ---
Write-Host "Setting MISTRAL_API_KEY environment variable (User Scope)..." -ForegroundColor Yellow
# setx is external and creates a persistent variable, but it does not update the current session
setx MISTRAL_API_KEY "$API_KEY" /M
# Also set for the current session for immediate use
$env:MISTRAL_API_KEY = $API_KEY
Write-Host "API Key set." -ForegroundColor Green

# --- 7. Build and install pdftomd ---
Write-Host "Building pdftomd executable..." -ForegroundColor Yellow
go build -o pdftomd.exe pdftomd.go
if ($LASTEXITCODE -ne 0) {
    Write-Host "Error: Go build failed. Check pdftomd.go for errors." -ForegroundColor Red
    exit 1
}
Write-Host "Build successful." -ForegroundColor Green

Write-Host "Moving pdftomd.exe to $TOOLS_DIR" -ForegroundColor Yellow
Move-Item -Path pdftomd.exe -Destination $TOOLS_DIR -Force
if ($LASTEXITCODE -ne 0) {
    Write-Host "Error: Failed to move executable." -ForegroundColor Red
    exit 1
}

# --- 8. Verification ---
Write-Host "" -ForegroundColor Cyan
Write-Host "--- Installation Complete! ---" -ForegroundColor Cyan
Write-Host "The 'nanocode' and 'pdftomd' commands are now installed." -ForegroundColor Cyan
Write-Host "NOTE: You must open a new PowerShell window for the PATH changes to take effect." -ForegroundColor Yellow
Write-Host "You can test it now in the NEW window by running: $EXEC_NAME" -ForegroundColor Cyan
Write-Host "" -ForegroundColor Cyan