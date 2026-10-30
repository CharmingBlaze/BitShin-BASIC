# IDE build. The window is Neutralino, not Wails, so there is no Wails build-tag dialog.
# https://neutralino.js.org/docs/configuration/neutralino.config.json

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

Write-Host "1. Building moon.exe and moonsls.exe..."
go build -o moon.exe ./cmd/bs
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
go build -o moonsls.exe ./cmd/bsls
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

$dllDir = Join-Path $root "third_party\windows"
if (Test-Path $dllDir) {
    Copy-Item (Join-Path $dllDir "*.dll") -Destination $root -Force
}

$neu = Join-Path $root "ide\bin\neutralino-win_x64.exe"
if (-not (Test-Path $neu)) {
    Write-Host "2. Downloading Neutralino 6.10.0..."
    $zip = Join-Path $env:TEMP "neutralinojs-v6.10.0.zip"
    $unpack = Join-Path $env:TEMP "neutralinojs-v6.10.0"
    Invoke-WebRequest -Uri "https://github.com/neutralinojs/neutralinojs/releases/download/v6.10.0/neutralinojs-v6.10.0.zip" -OutFile $zip
    if (Test-Path $unpack) { Remove-Item $unpack -Recurse -Force }
    Expand-Archive -Path $zip -DestinationPath $unpack -Force
    $found = Get-ChildItem $unpack -Recurse -Filter "neutralino-win_x64.exe" | Select-Object -First 1
    if (-not $found) { throw "neutralino-win_x64.exe was not in the Neutralino zip" }
    New-Item -ItemType Directory -Force -Path (Split-Path $neu) | Out-Null
    Copy-Item $found.FullName $neu -Force
}

Write-Host "3. Building IDE frontend..."
Set-Location (Join-Path $root "ide\frontend")
if (-not (Test-Path "node_modules")) {
    npm install
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
}
npm run build
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

Write-Host "4. Compiling IDE..."
Set-Location (Join-Path $root "ide")
go build -o (Join-Path $root "moonbasic.exe") .
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
Copy-Item $neu (Join-Path $root "neutralino-win_x64.exe") -Force
Copy-Item (Join-Path $root "ide\neutralino.config.json") (Join-Path $root "neutralino.config.json") -Force

Write-Host "Built moonbasic.exe"
