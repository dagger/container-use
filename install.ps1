#Requires -Version 5.0
<#
.Description
    Download and install container-use.

.PARAMETER Version
    Version of container-use to install (e.g., v0.4.0). Defaults to latest.
    Also supports a 7-40 char git commit or "latest".

.PARAMETER DownloadPath
    Temporary download location seed. The script will place artifacts in the same directory
    as this temp file. Defaults to a system temp file.

.PARAMETER InstallPath
    Installation directory. Defaults to $env:USERPROFILE\container-use

.PARAMETER AddToPath
    If set, add the installation directory to the user's PATH (no elevation required).

.PARAMETER Repo
    (Advanced) GitHub "owner/repo" to install from. Defaults to dagger/container-use.
    Useful for testing a fork without editing the script.

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

.EXAMPLE
    # Install from a fork's releases for testing
    .\install.ps1 -Repo "grouville/container-use" -Version v0.3.5-test -AddToPath
#>

Param (
    [Parameter(Mandatory = $false)]
    [ValidatePattern('^(latest|v?\d+\.\d+\.\d+(?:-[A-Za-z0-9.-]+)?|[a-f0-9]{7,40})$')]
    [string]$Version = "latest",

    [Parameter(Mandatory = $false)]
    [ValidateNotNullOrEmpty()]
    [string]$DownloadPath = [System.IO.Path]::GetTempFileName(),

    [Parameter(Mandatory = $false)]
    [ValidateNotNullOrEmpty()]
    [string]$InstallPath = "$env:USERPROFILE\container-use",

    [Parameter(Mandatory = $false)]
    [switch]$AddToPath = $false,

    [Parameter(Mandatory = $false)]
    [ValidatePattern('^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$')]
    [string]$Repo = "dagger/container-use"
)

# ---------------------------------------------------------------------------------
# Container Use Installation Utility for Windows
# Hardened for PS 5.1 and PS 7+; secure downloads; robust PATH handling
# ---------------------------------------------------------------------------------

$ErrorActionPreference = "Stop"

# Ensure TLS 1.2 (older Windows can default to TLS 1.0/1.1)
try { [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12 } catch { }

# Configuration
$REPO = $Repo
$BINARY_NAME = "container-use"

# Conditionally supply -UseBasicParsing (PS < 6 supports it; PS 7+ removed it)
function New-WebArgs {
    param([Parameter(Mandatory=$true)] [string]$Uri)
    $args = @{ Uri = $Uri }
    if ($PSVersionTable.PSVersion.Major -lt 6) { $args.UseBasicParsing = $true }
    return $args
}

# Safely append to PATH (User by default), avoiding dupes and empty tails
function Add-PathEntry {
    param(
        [Parameter(Mandatory=$true)] [string]$PathToAdd,
        [Parameter(Mandatory=$false)] [ValidateSet('User','Machine')] [string]$Scope = 'User'
    )
    $cur = [Environment]::GetEnvironmentVariable('Path', $Scope)
    if (-not $cur) { $cur = '' }
    $parts = @($cur -split ';' | ForEach-Object { $_.Trim() } | Where-Object { $_ }) + @()

    if ($parts -notcontains $PathToAdd) {
        $newPath = ($parts + $PathToAdd) -join ';'
        [Environment]::SetEnvironmentVariable('Path', $newPath, $Scope)
        Write-Host "Added to $Scope PATH. Restart your terminal to pick it up." -ForegroundColor Green
    } else {
        Write-Host "Path already contains $PathToAdd" -ForegroundColor Green
    }
}

function Get-ProcessorArchitecture {
    # Map to Go arch names we ship: amd64, arm64
    $arch = $env:PROCESSOR_ARCHITECTURE
    switch ($arch) {
        'AMD64' { return 'amd64' }
        'ARM64' { return 'arm64' }
        default { throw "Unsupported architecture: $arch (supported: AMD64, ARM64)" }
    }
}

function Find-LatestVersion {
    try {
        $release = Invoke-RestMethod @((New-WebArgs "https://api.github.com/repos/$REPO/releases/latest")) -UserAgent "PowerShell"
        return $release.tag_name
    } catch {
        throw "Failed to fetch latest release: $_"
    }
}

function Get-DownloadUrl {
    Param (
        [Parameter(Mandatory = $true)] [string]$Version,
        [Parameter(Mandatory = $true)] [string]$Arch
    )
    # GoReleaser uses the tag (with 'v') in archive names
    "https://github.com/$REPO/releases/download/$Version/container-use_${Version}_windows_${Arch}.zip"
}

function Get-ChecksumUrl {
    Param ([Parameter(Mandatory = $true)] [string]$Version)
    "https://github.com/$REPO/releases/download/$Version/checksums.txt"
}

function Get-Checksum {
    Param (
        [Parameter(Mandatory = $true)] [string]$Version,
        [Parameter(Mandatory = $true)] [string]$Arch
    )
    $checksumUrl = Get-ChecksumUrl -Version $Version
    $target = "container-use_${Version}_windows_${Arch}.zip"

    try {
        $response = Invoke-RestMethod @((New-WebArgs $checksumUrl)) -UserAgent "PowerShell"
        $checksums = $response -split "`n"

        foreach ($line in $checksums) {
            if ($line -match [regex]::Escape($target)) {
                return ($line -split ' ' | Select-Object -First 1)
            }
        }
        throw "Checksum not found for $target"
    } catch {
        throw "Failed to fetch or parse checksums: $_"
    }
}

function Compare-Checksum {
    Param (
        [Parameter(Mandatory = $true)] [string]$FilePath,
        [Parameter(Mandatory = $true)] [string]$ExpectedChecksum
    )
    $hash = (Get-FileHash -Path $FilePath -Algorithm SHA256).Hash
    if ($hash.ToUpperInvariant() -ne $ExpectedChecksum.ToUpperInvariant()) {
        Remove-Item -Path $FilePath -Force
        throw "Checksum mismatch. Expected: $ExpectedChecksum, Got: $hash"
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

    if (-not (Get-Command docker -ErrorAction SilentlyContinue)) {
        Write-Host "  Docker is required but not installed." -ForegroundColor Red
        Write-Host "  Install Docker Desktop: https://docs.docker.com/desktop/install/windows-install/" -ForegroundColor Yellow
        return $false
    }
    if (-not (Get-Command git -ErrorAction SilentlyContinue)) {
        Write-Host "  Git is required but not installed." -ForegroundColor Red
        Write-Host "  Install Git: https://git-scm.com/download/win" -ForegroundColor Yellow
        return $false
    }

    try { docker version *>$null } catch { Write-Host "  Docker is installed but not responding." -ForegroundColor Red; return $false }
    try { git version    *>$null } catch { Write-Host "  Git is installed but not responding."    -ForegroundColor Red; return $false }

    Write-Host "  Docker is installed" -ForegroundColor Green
    Write-Host "  Git is installed"    -ForegroundColor Green
    return $true
}

function Install-ContainerUse {
    Write-Host ""
    Write-Host "Container Use Installer for Windows" -ForegroundColor Cyan
    Write-Host "===================================" -ForegroundColor Cyan
    Write-Host ""

    if (-not (Test-Dependencies)) {
        throw "Missing required dependencies"
    }

    $targetVersion = $Version
    if ($targetVersion -eq "latest") {
        Write-Host "Finding latest version..." -ForegroundColor Blue
        $targetVersion = Find-LatestVersion
        Write-Host "Latest version: $targetVersion" -ForegroundColor Green
    }

    $arch = Get-ProcessorArchitecture
    Write-Host "Architecture: $arch" -ForegroundColor Blue

    $downloadUrl = Get-DownloadUrl -Version $targetVersion -Arch $arch

    $zipName = "container-use_${targetVersion}_windows_${arch}.zip"
    $zipPath = [System.IO.Path]::Combine([System.IO.Path]::GetDirectoryName($DownloadPath), $zipName)

    Write-Host "Downloading from $downloadUrl..." -ForegroundColor Blue
    try {
        Invoke-WebRequest @((New-WebArgs $downloadUrl)) -OutFile $zipPath
        Write-Host "Downloaded successfully" -ForegroundColor Green
    } catch {
        throw "Failed to download: $_"
    }

    Write-Host "Verifying checksum..." -ForegroundColor Blue
    try {
        $expectedChecksum = Get-Checksum -Version $targetVersion -Arch $arch
        Compare-Checksum -FilePath $zipPath -ExpectedChecksum $expectedChecksum
        Write-Host "Checksum verified" -ForegroundColor Green
    } catch {
        throw "Checksum verification failed: $_"
    }

    $tempExtractPath = [System.IO.Path]::Combine([System.IO.Path]::GetTempPath(), "container-use-extract-$(Get-Random)")
    Write-Host "Extracting..." -ForegroundColor Blue
    try {
        Expand-Archive -Path $zipPath -DestinationPath $tempExtractPath -Force
    } catch {
        throw "Failed to extract: $_"
    } finally {
        Remove-Item $zipPath -Force -ErrorAction SilentlyContinue
    }

    $installFullPath = Get-InstallPath

    $exePath = Join-Path $tempExtractPath "container-use.exe"
    if (-not (Test-Path $exePath)) {
        throw "container-use.exe not found in archive"
    }

    $destPath = Join-Path $installFullPath "container-use.exe"
    Copy-Item -Path $exePath -Destination $destPath -Force
    Write-Host "Installed to $destPath" -ForegroundColor Green

    # Create cu.exe alias for convenience
    $cuPath = Join-Path $installFullPath "cu.exe"
    Copy-Item -Path $destPath -Destination $cuPath -Force
    Write-Host "Created cu.exe alias" -ForegroundColor Green

    # Cleanup
    Remove-Item $tempExtractPath -Recurse -Force -ErrorAction SilentlyContinue

    # PATH update on request
    if ($AddToPath) {
        Add-PathEntry -PathToAdd $installFullPath -Scope 'User'
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
        $versionOutput = (& $destPath version 2>&1 | Out-String).Trim()
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
