$ErrorActionPreference = "Stop"
Set-Location (Split-Path -Parent $PSScriptRoot)
go build -o moon.exe ./cmd/moon
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
go build -o moonsls.exe ./cmd/moonsls
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
$dllDir = Join-Path (Get-Location) "third_party\windows"
if (Test-Path $dllDir) {
    Copy-Item (Join-Path $dllDir "*.dll") -Destination . -Force
}
Write-Host "Built moon.exe and moonsls.exe (Windows DLLs copied next to the exe)"
