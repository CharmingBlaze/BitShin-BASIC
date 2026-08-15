$ErrorActionPreference = "Stop"
Set-Location (Split-Path -Parent $PSScriptRoot)

Write-Host "1. Building BitShin BASIC CLI..."
go build -o bs.exe ./cmd/bs
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

$dllDir = Join-Path (Get-Location) "third_party\windows"
if (Test-Path $dllDir) {
    Copy-Item (Join-Path $dllDir "*.dll") -Destination . -Force
}

Write-Host "2. Building Wails Svelte 5 IDE Application..."
Set-Location "ide"
wails build
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

Set-Location (Split-Path -Parent $PSScriptRoot)
Copy-Item "ide/build/bin/BitShinIDE.exe" -Destination . -Force

Write-Host "Built BitShinIDE.exe successfully!"
