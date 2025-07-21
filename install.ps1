#Requires -Version 5.0

<#
.Description
    Download and install container-use.
.PARAMETER Version
    Version of container-use to install (e.g., v0.4.0). Defaults to latest.
.PARAMETER InstallPath
    Installation directory. Defaults to $env:USERPROFILE\container-use
.PARAMETER DownloadPath
    Temporary download location. Defaults to temp file.
.PARAMETER AddToPath
    Add installation directory to PATH.
.EXAMPLE
    .\install.ps1
    Install latest version with default settings.
.EXAMPLE
    .\install.ps1 -InstallPath "C:\tools\container-use"
    Install to C:\tools\container-use.
.EXAMPLE
    .\install.ps1 -Version v0.4.0
    Install specified version v0.4.0.
.EXAMPLE
    .\install.ps1 -AddToPath
    Install and add to PATH.
#>

Param (
    [Parameter(Mandatory = $false)][string]$Version = "latest",
    [Parameter(Mandatory = $false)][string]$DownloadPath = [System.IO.Path]::GetTempFileName(),
    [Parameter(Mandatory = $false)][string]$InstallPath = "$env:USERPROFILE\container-use",
    [Parameter(Mandatory = $false)][switch]$AddToPath = $false
)

# ---------------------------------------------------------------------------------
# Container Use Installation Utility for Windows
# Based on Dagger's installation script
# ---------------------------------------------------------------------------------

$ErrorActionPreference = "Stop"

# Configuration
$REPO = "dagger/container-use"
$BINARY_NAME = "container-use"

# Helper functions
function Write-Info {
    param([string]$Message)
    Write-Host "ℹ️  $Message" -ForegroundColor Blue
}

function Write-Success {
    param([string]$Message)
    Write-Host "✅ $Message" -ForegroundColor Green
}

function Write-Warning {
    param([string]$Message)
    Write-Host "⚠️  $Message" -ForegroundColor Yellow
}

function Write-Error {
    param([string]$Message)
    Write-Host "❌ $Message" -ForegroundColor Red
}

function Get-ProcessorArchitecture {
    $arch = $env:PROCESSOR_ARCHITECTURE
    switch ($arch) {
        "AMD64" { return "amd64" }
        "ARM64" { return "arm64" }
        default {
            throw "Unsupported architecture: $arch"
        }
    }
}

function Test-Dependencies {
    Write-Info "Checking dependencies..."
    
    # Check Docker
    try {
        $null = docker version 2>&1
        Write-Success "Docker is installed"
    } catch {
        Write-Error "Docker is required but not installed."
        Write-Info "Please install Docker Desktop from: https://docs.docker.com/desktop/install/windows-install/"
        return $false
    }
    
    # Check Git
    try {
        $null = git version 2>&1
        Write-Success "Git is installed"
    } catch {
        Write-Error "Git is required but not installed."
        Write-Info "Please install Git from: https://git-scm.com/download/win"
        return $false
    }
    
    return $true
}

function Get-LatestVersion {
    try {
        $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$REPO/releases/latest" -UseBasicParsing
        return $release.tag_name
    } catch {
        throw "Failed to fetch latest release: $_"
    }
}

function Get-DownloadUrl {
    param(
        [string]$Version,
        [string]$Arch
    )
    
    # Clean version (remove 'v' prefix if present)
    $cleanVersion = $Version -replace '^v', ''
    
    # Construct filename based on GoReleaser output
    $fileName = "container-use_${cleanVersion}_windows_${Arch}.zip"
    return "https://github.com/$REPO/releases/download/$Version/$fileName"
}

function Install-ContainerUse {
    # Get version
    $targetVersion = $Version
    if ($targetVersion -eq "latest") {
        $targetVersion = Get-LatestVersion
        Write-Info "Latest version: $targetVersion"
    }
    
    # Get architecture
    $arch = Get-ProcessorArchitecture
    Write-Info "Architecture: $arch"
    
    # Get download URL
    $downloadUrl = Get-DownloadUrl -Version $targetVersion -Arch $arch
    
    # Download
    $zipName = "container-use_$($targetVersion -replace '^v', '')_windows_$arch.zip"
    $zipPath = [System.IO.Path]::Combine([System.IO.Path]::GetDirectoryName($DownloadPath), $zipName)
    
    Write-Info "Downloading $downloadUrl..."
    try {
        Invoke-WebRequest -Uri $downloadUrl -OutFile $zipPath -UseBasicParsing
        Write-Success "Downloaded to $zipPath"
    } catch {
        Write-Error "Failed to download: $_"
        exit 1
    }
    
    # Extract
    $tempExtractPath = [System.IO.Path]::Combine([System.IO.Path]::GetTempPath(), "container-use-extract-$(Get-Random)")
    Write-Info "Extracting..."
    try {
        Expand-Archive -Path $zipPath -DestinationPath $tempExtractPath -Force
    } catch {
        Write-Error "Failed to extract: $_"
        exit 1
    } finally {
        Remove-Item $zipPath -Force -ErrorAction SilentlyContinue
    }
    
    # Install
    if (-not (Test-Path $InstallPath)) {
        New-Item -ItemType Directory -Path $InstallPath -Force | Out-Null
    }
    
    $exePath = Join-Path $tempExtractPath "container-use.exe"
    if (-not (Test-Path $exePath)) {
        Write-Error "container-use.exe not found in archive"
        exit 1
    }
    
    $destPath = Join-Path $InstallPath "container-use.exe"
    Copy-Item -Path $exePath -Destination $destPath -Force
    Write-Success "Installed to $destPath"
    
    # Create cu.exe alias
    $cuPath = Join-Path $InstallPath "cu.exe"
    Copy-Item -Path $destPath -Destination $cuPath -Force
    Write-Success "Created cu.exe alias"
    
    # Cleanup
    Remove-Item $tempExtractPath -Recurse -Force -ErrorAction SilentlyContinue
}

function Update-Path {
    if ($AddToPath) {
        $userPath = [Environment]::GetEnvironmentVariable("Path", [EnvironmentVariableTarget]::User)
        if ($userPath -notlike "*$InstallPath*") {
            Write-Info "Adding $InstallPath to user PATH..."
            $newPath = "$userPath;$InstallPath"
            [Environment]::SetEnvironmentVariable("Path", $newPath, [EnvironmentVariableTarget]::User)
            Write-Success "Added to PATH (restart your terminal to use)"
        } else {
            Write-Success "$InstallPath is already in PATH"
        }
    } else {
        Write-Info "To add container-use to your PATH, run:"
        Write-Host "    `$env:Path += `";$InstallPath`"" -ForegroundColor Yellow
        Write-Host "    Or run this script again with -AddToPath" -ForegroundColor Yellow
    }
}

# Main
try {
    Write-Host ""
    Write-Host "Container Use Installer for Windows" -ForegroundColor Cyan
    Write-Host "===================================" -ForegroundColor Cyan
    Write-Host ""
    
    # Check dependencies
    if (-not (Test-Dependencies)) {
        Write-Error "Missing required dependencies"
        exit 1
    }
    
    # Install
    Install-ContainerUse
    
    # Update PATH
    Update-Path
    
    # Verify installation
    Write-Info "Verifying installation..."
    $exePath = Join-Path $InstallPath "container-use.exe"
    if (Test-Path $exePath) {
        try {
            $versionOutput = & $exePath version 2>&1
            Write-Success "container-use is ready! Version: $versionOutput"
        } catch {
            Write-Warning "container-use installed but couldn't verify version"
        }
    }
    
    Write-Host ""
    Write-Success "Installation complete!"
    Write-Info "Run 'container-use --help' to get started"
    if (-not $AddToPath) {
        Write-Info "Remember to add $InstallPath to your PATH"
    }
    Write-Host ""
} catch {
    Write-Error $_.Exception.Message
    exit 1
}