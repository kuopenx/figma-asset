#Requires -Version 5.1
$ErrorActionPreference = "Stop"

$Owner = "kuopenx"
$Repo = "figma-asset"
$BinDir = "$env:LOCALAPPDATA\figma-asset"
$PluginDir = "$env:USERPROFILE\figma-asset-plugin"

# Detect platform.
$Arch = $env:PROCESSOR_ARCHITECTURE
switch ($Arch) {
    "AMD64" { $Arch = "amd64" }
    "ARM64" { $Arch = "arm64" }
    default { Write-Error "Unsupported architecture: $Arch"; exit 1 }
}

$ZipName = "figma-asset-windows-$Arch.zip"
$DownloadUrl = "https://github.com/$Owner/$Repo/releases/latest/download/$ZipName"
$ChecksumsUrl = "https://github.com/$Owner/$Repo/releases/latest/download/checksums.txt"

$TmpDir = New-Item -ItemType Directory -Force -Path (Join-Path $env:TEMP "figma-asset-install-$(Get-Random)")

Write-Host "Detected platform: windows/$Arch"
Write-Host "Downloading $ZipName..."

# Download zip.
$ZipPath = Join-Path $TmpDir.FullName $ZipName
Invoke-WebRequest -Uri $DownloadUrl -OutFile $ZipPath

# Download checksums and verify.
$ChecksumsPath = Join-Path $TmpDir.FullName "checksums.txt"
Invoke-WebRequest -Uri $ChecksumsUrl -OutFile $ChecksumsPath

$Expected = (Get-Content $ChecksumsPath | Where-Object { $_ -match $ZipName } | ForEach-Object { ($_.Trim() -split '\s+')[0] } | Select-Object -First 1)
if (-not $Expected) {
    Write-Error "No checksum found for $ZipName"
    exit 1
}

$Actual = (Get-FileHash $ZipPath -Algorithm SHA256).Hash.ToLower()
if ($Actual -ne $Expected.ToLower()) {
    Write-Error "Checksum mismatch: expected $Expected, got $Actual"
    exit 1
}
Write-Host "Checksum verified."

# Extract.
Expand-Archive -Path $ZipPath -DestinationPath (Join-Path $TmpDir.FullName "extract") -Force

# Install binary.
New-Item -ItemType Directory -Force -Path $BinDir | Out-Null
Copy-Item -Path (Join-Path $TmpDir.FullName "extract\figma-asset.exe") -Destination $BinDir -Force

# Install plugin files.
New-Item -ItemType Directory -Force -Path $PluginDir | Out-Null
Copy-Item -Path (Join-Path $TmpDir.FullName "extract\manifest.json") -Destination $PluginDir -Force
$ExtractedPlugin = Join-Path $TmpDir.FullName "extract\plugin"
if (Test-Path $ExtractedPlugin) {
    $PluginSubDir = Join-Path $PluginDir "plugin"
    New-Item -ItemType Directory -Force -Path $PluginSubDir | Out-Null
    Get-ChildItem $ExtractedPlugin | ForEach-Object {
        Copy-Item $_.FullName -Destination $PluginSubDir -Force
    }
}

Write-Host ""
Write-Host "========================================="
Write-Host "  Installation complete!"
Write-Host "========================================="
Write-Host ""
Write-Host "Binary location:"
Write-Host "  $BinDir\figma-asset.exe"
Write-Host ""
Write-Host "Plugin location:"
Write-Host "  $PluginDir\manifest.json"
Write-Host "  $PluginDir\plugin\code.js"
Write-Host "  $PluginDir\plugin\ui.html"
Write-Host ""

# PATH check.
$userPath = [Environment]::GetEnvironmentVariable("PATH", "User")
if ($userPath -notlike "*$BinDir*") {
    Write-Host "-----------------------------------------"
    Write-Host "  ACTION REQUIRED: Add to PATH"
    Write-Host "-----------------------------------------"
    Write-Host "  Run this command to add to PATH:"
    Write-Host ""
    Write-Host "    [Environment]::SetEnvironmentVariable('PATH', `$env:PATH + ';$BinDir', 'User')"
    Write-Host ""
    Write-Host "  Then restart your terminal."
    Write-Host ""
}

Write-Host "-----------------------------------------"
Write-Host "  Import plugin into Figma (one-time)"
Write-Host "-----------------------------------------"
Write-Host "  1. Open Figma Desktop"
Write-Host "  2. Menu: Plugins -> Development -> Import plugin from manifest..."
Write-Host "  3. Select: $PluginDir\manifest.json"
Write-Host ""
Write-Host "  After import, run:"
Write-Host "    figma-asset start"
Write-Host "    figma-asset export png --platform flutter --node <node-id> --project-dir ~\my_app"
Write-Host ""
Write-Host "  To check for updates:"
Write-Host "    figma-asset upgrade --check"
Write-Host "  To update:"
Write-Host "    figma-asset upgrade"