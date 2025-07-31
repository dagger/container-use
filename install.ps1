#Requires -Version 5.0

<#
.Description
    Download and install container-use.
.PARAMETER Version
    Version of container-use to install (e.g., v0.4.0). Defaults to latest.
.PARAMETER DownloadPath
    Temporary download location. Defaults to temp file.
.PARAMETER InstallPath
    Installation directory. Defaults to $env:USERPROFILE\container-use
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
    [Parameter(Mandatory = $false)]
    [ValidatePattern('^(latest|v?\d+\.\d+\.\d+(-[a-zA-Z0-9]+)?|[a-f0-9]{7,40})$')]
    [string]$Version = "latest",
    
    [Parameter(Mandatory = $false)]
    [ValidateNotNullOrEmpty()]
    [string]$DownloadPath = [System.IO.Path]::GetTempFileName(),
    
    [Parameter(Mandatory = $false)]
    [ValidateNotNullOrEmpty()]
    [string]$InstallPath = "$env:USERPROFILE\container-use",
    
    [Parameter(Mandatory = $false)]
    [switch]$AddToPath = $false
)

# ---------------------------------------------------------------------------------
# Container Use Installation Utility for Windows
# Based on Dagger's installation script
# ---------------------------------------------------------------------------------

$ErrorActionPreference = "Stop"

# Configuration
$REPO = "dagger/container-use"
$BINARY_NAME = "container-use"

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

function Find-LatestVersion {
    try {
        $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$REPO/releases/latest" -UseBasicParsing
        return $release.tag_name
    } catch {
        throw "Failed to fetch latest release: $_"
    }
}

function Get-DownloadUrl {
    Param (
        [Parameter(Mandatory = $true)]
        [string]$Version,
        [Parameter(Mandatory = $true)]
        [string]$Arch
    )
    
    # GoReleaser uses full tag (with v) in archive names
    return "https://github.com/$REPO/releases/download/$Version/container-use_${Version}_windows_${Arch}.zip"
}

function Get-ChecksumUrl {
    Param (
        [Parameter(Mandatory = $true)]
        [string]$Version
    )
    
    return "https://github.com/$REPO/releases/download/$Version/checksums.txt"
}

function Get-Checksum {
    Param (
        [Parameter(Mandatory = $true)]
        [string]$Version,
        [Parameter(Mandatory = $true)]
        [string]$Arch
    )
    
    $checksumUrl = Get-ChecksumUrl -Version $Version
    # GoReleaser uses full tag (with v) in archive names
    $target = "container-use_${Version}_windows_${Arch}.zip"
    
    try {
        $response = Invoke-RestMethod -Uri $checksumUrl -UserAgent "PowerShell"
        $checksums = $response -split "`n"
        
        $checksum = $null
        foreach ($line in $checksums) {
            if ($line -match $target) {
                $checksum = $line -split " " | Select-Object -First 1
                break
            }
        }
        
        if ([string]::IsNullOrWhiteSpace($checksum)) {
            throw "Checksum not found for $target"
        }
        
        return $checksum
    } catch {
        throw "Failed to fetch or parse checksums: $_"
    }
}

function Compare-Checksum {
    Param (
        [Parameter(Mandatory = $true)]
        [string]$FilePath,
        [Parameter(Mandatory = $true)]
        [string]$ExpectedChecksum
    )
    
    $hash = Get-FileHash -Path $FilePath -Algorithm SHA256
    
    if ($hash.Hash -ne $ExpectedChecksum) {
        Remove-Item -Path $FilePath -Force
        throw "Checksum mismatch. Expected: $ExpectedChecksum, Got: $($hash.Hash)"
    }
}

function Get-InstallPath {
    if (-not (Test-Path $InstallPath)) {
        New-Item -ItemType Directory -Path $InstallPath -Force | Out-Null
    }
    return (Get-Item -Path $InstallPath).FullName
}

function Test-Dependencies {
    Write-Host "Checking dependencies..." -ForegroundColor Blue
    
    # Check Docker
    try {
        $null = docker version 2>&1
        Write-Host "  Docker is installed" -ForegroundColor Green
    } catch {
        Write-Host "  Docker is required but not installed." -ForegroundColor Red
        Write-Host "  Please install Docker Desktop from: https://docs.docker.com/desktop/install/windows-install/" -ForegroundColor Yellow
        return $false
    }
    
    # Check Git
    try {
        $null = git version 2>&1
        Write-Host "  Git is installed" -ForegroundColor Green
    } catch {
        Write-Host "  Git is required but not installed." -ForegroundColor Red
        Write-Host "  Please install Git from: https://git-scm.com/download/win" -ForegroundColor Yellow
        return $false
    }
    
    return $true
}

function Install-ContainerUse {
    Write-Host ""
    Write-Host "Container Use Installer for Windows" -ForegroundColor Cyan
    Write-Host "===================================" -ForegroundColor Cyan
    Write-Host ""
    
    # Check dependencies
    if (-not (Test-Dependencies)) {
        throw "Missing required dependencies"
    }
    
    # Get version
    $targetVersion = $Version
    if ($targetVersion -eq "latest") {
        Write-Host "Finding latest version..." -ForegroundColor Blue
        $targetVersion = Find-LatestVersion
        Write-Host "Latest version: $targetVersion" -ForegroundColor Green
    }
    
    # Get architecture
    $arch = Get-ProcessorArchitecture
    Write-Host "Architecture: $arch" -ForegroundColor Blue
    
    # Get download URL
    $downloadUrl = Get-DownloadUrl -Version $targetVersion -Arch $arch
    
    # Download
    $zipName = "container-use_${targetVersion}_windows_${arch}.zip"
    $zipPath = [System.IO.Path]::Combine([System.IO.Path]::GetDirectoryName($DownloadPath), $zipName)
    
    Write-Host "Downloading from $downloadUrl..." -ForegroundColor Blue
    try {
        Invoke-WebRequest -Uri $downloadUrl -OutFile $zipPath -UseBasicParsing
        Write-Host "Downloaded successfully" -ForegroundColor Green
    } catch {
        throw "Failed to download: $_"
    }
    
    # Verify checksum
    Write-Host "Verifying checksum..." -ForegroundColor Blue
    try {
        $expectedChecksum = Get-Checksum -Version $targetVersion -Arch $arch
        Compare-Checksum -FilePath $zipPath -ExpectedChecksum $expectedChecksum
        Write-Host "Checksum verified" -ForegroundColor Green
    } catch {
        throw "Checksum verification failed: $_"
    }
    
    # Extract
    $tempExtractPath = [System.IO.Path]::Combine([System.IO.Path]::GetTempPath(), "container-use-extract-$(Get-Random)")
    Write-Host "Extracting..." -ForegroundColor Blue
    try {
        Expand-Archive -Path $zipPath -DestinationPath $tempExtractPath -Force
    } catch {
        throw "Failed to extract: $_"
    } finally {
        Remove-Item $zipPath -Force -ErrorAction SilentlyContinue
    }
    
    # Install
    $installFullPath = Get-InstallPath
    
    $exePath = Join-Path $tempExtractPath "container-use.exe"
    if (-not (Test-Path $exePath)) {
        throw "container-use.exe not found in archive"
    }
    
    $destPath = Join-Path $installFullPath "container-use.exe"
    Copy-Item -Path $exePath -Destination $destPath -Force
    Write-Host "Installed to $destPath" -ForegroundColor Green
    
    # Create cu.exe alias
    $cuPath = Join-Path $installFullPath "cu.exe"
    Copy-Item -Path $destPath -Destination $cuPath -Force
    Write-Host "Created cu.exe alias" -ForegroundColor Green
    
    # Cleanup
    Remove-Item $tempExtractPath -Recurse -Force -ErrorAction SilentlyContinue
    
    # Update PATH
    if ($AddToPath) {
        $userPath = [Environment]::GetEnvironmentVariable("Path", [EnvironmentVariableTarget]::User)
        if ($userPath -notlike "*$installFullPath*") {
            Write-Host "Adding $installFullPath to user PATH..." -ForegroundColor Blue
            # Ensure path doesn't end with semicolon before appending
            $separator = if ($userPath -and $userPath[-1] -ne ';') { ';' } else { '' }
            $newPath = "$userPath$separator$installFullPath"
            [Environment]::SetEnvironmentVariable("Path", $newPath, [EnvironmentVariableTarget]::User)
            Write-Host "Added to PATH (restart your terminal to use)" -ForegroundColor Green
        } else {
            Write-Host "$installFullPath is already in PATH" -ForegroundColor Green
        }
    } else {
        Write-Host ""
        Write-Host "To add container-use to your PATH, run:" -ForegroundColor Yellow
        Write-Host "    [Environment]::SetEnvironmentVariable('Path', `$env:Path + ';$installFullPath', [EnvironmentVariableTarget]::User)" -ForegroundColor White
        Write-Host "Or run this script again with -AddToPath" -ForegroundColor Yellow
    }
    
    # Verify installation
    Write-Host ""
    Write-Host "Verifying installation..." -ForegroundColor Blue
    try {
        $versionOutput = & $destPath version 2>&1
        Write-Host "container-use is ready! Version: $versionOutput" -ForegroundColor Green
    } catch {
        Write-Host "container-use installed but couldn't verify version" -ForegroundColor Yellow
    }
    
    Write-Host ""
    Write-Host "Installation complete!" -ForegroundColor Green
    Write-Host "Run 'container-use --help' to get started" -ForegroundColor Cyan
}

# Main execution
try {
    Install-ContainerUse
} catch {
    Write-Host ""
    Write-Host "Installation failed: $_" -ForegroundColor Red
    exit 1
}