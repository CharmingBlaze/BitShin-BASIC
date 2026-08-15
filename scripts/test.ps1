$ErrorActionPreference = "Stop"
Set-Location (Split-Path -Parent $PSScriptRoot)
go test ./internal/...
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
go vet ./internal/lex ./internal/parse ./internal/interp ./internal/value ./internal/ast ./internal/phys2d ./internal/phys3d ./internal/netenet
exit $LASTEXITCODE
