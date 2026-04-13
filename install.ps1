#Requires -Version 5.1
<#
.SYNOPSIS
    Installs the bnm command-line tool on Windows.
.PARAMETER Version
    The version to install (e.g. "v0.1.0"). Defaults to the latest release.
.PARAMETER InstallDir
    Directory to install bnm.exe. Defaults to "$env:LOCALAPPDATA\bnm".
.EXAMPLE
    irm https://raw.githubusercontent.com/Yarnadori/bnm/main/install.ps1 | iex
.EXAMPLE
    .\install.ps1 -Version v0.1.0
#>
param(
    [string]$Version = "",
    [string]$InstallDir = "$env:LOCALAPPDATA\bnm"
)

$ErrorActionPreference = "Stop"
$Repo = "Yarnadori/bnm"
$BinaryName = "bnm.exe"

# Detect architecture
$arch = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture
if ($arch -eq [System.Runtime.InteropServices.Architecture]::Arm64) {
    $AssetName = "bnm-windows-arm64.exe"
} else {
    $AssetName = "bnm-windows-amd64.exe"
}

# Determine version
if (-not $Version) {
    Write-Host "Fetching latest release version..."
    $releaseInfo = Invoke-RestMethod "https://api.github.com/repos/$Repo/releases/latest"
    $Version = $releaseInfo.tag_name
}

if (-not $Version) {
    Write-Error "Failed to determine the latest version."
    exit 1
}

$DownloadUrl = "https://github.com/$Repo/releases/download/$Version/$AssetName"
$Destination = Join-Path $InstallDir $BinaryName

Write-Host "Installing bnm $Version..."
Write-Host "Downloading from: $DownloadUrl"

# Create install directory if it doesn't exist
if (-not (Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir | Out-Null
}

# Download binary
Invoke-WebRequest -Uri $DownloadUrl -OutFile $Destination -UseBasicParsing

Write-Host "bnm $Version installed to $Destination"

# Add to PATH for current user if not already present
$userPath = [Environment]::GetEnvironmentVariable("PATH", "User")
if ($userPath -notlike "*$InstallDir*") {
    [Environment]::SetEnvironmentVariable(
        "PATH",
        "$userPath;$InstallDir",
        "User"
    )
    Write-Host "Added $InstallDir to your PATH."
    Write-Host "Please restart your terminal for the PATH change to take effect."
} else {
    Write-Host "$InstallDir is already in your PATH."
}

Write-Host "Run 'bnm' to get started."
