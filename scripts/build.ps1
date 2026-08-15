$ErrorActionPreference = "Stop"
Set-Location (Split-Path -Parent $PSScriptRoot)
go build -o bs.exe ./cmd/bs
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
$dllDir = Join-Path (Get-Location) "third_party\windows"
if (Test-Path $dllDir) {
    Copy-Item (Join-Path $dllDir "*.dll") -Destination . -Force
}
Write-Host "Built bs.exe (Windows DLLs copied next to the exe)"
