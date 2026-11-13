# Release packaging: the moonBASIC SDK and the sample games, ready to zip and send.
#
#   .\scripts\package_sdk.ps1            # everything, into dist\
#   .\scripts\package_sdk.ps1 -NoZip     # folders only
#
# Writes:
#   dist\moonBASIC-<version>-windows-amd64\   compiler, language server, IDE, docs, examples, games
#   dist\Greenwood-<version>-windows\         standalone game (release profile)
#   dist\TheBlackVeil-<version>-windows\      standalone game (release profile)
#   and a .zip of each.
#
# Needs Go with a C compiler (MinGW-w64 gcc), and Node.js for the IDE frontend.
# The SDK itself needs neither: its moon.exe packages games without Go (see docs/RELEASE.md).

param([switch]$NoZip)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

$version = (Select-String -Path "cmd\moon\main.go" -Pattern 'const version = "([^"]+)"').Matches[0].Groups[1].Value
$distRoot = Join-Path $root "dist"
$sdkName = "moonBASIC-$version-windows-amd64"
$sdk = Join-Path $distRoot $sdkName

function Invoke-Checked([string]$what, [scriptblock]$block) {
    & $block
    if ($LASTEXITCODE -ne 0) { throw "$what failed (exit $LASTEXITCODE)" }
}

function Reset-Folder([string]$path) {
    # Only ever clears a folder inside dist\
    $full = [IO.Path]::GetFullPath($path)
    if (-not $full.StartsWith([IO.Path]::GetFullPath($distRoot) + [IO.Path]::DirectorySeparatorChar)) {
        throw "refusing to clear $full (outside dist)"
    }
    if (Test-Path $full) { Remove-Item $full -Recurse -Force }
    New-Item -ItemType Directory -Force -Path $full | Out-Null
}

# Copies a folder, leaving out test output and local files
function Copy-Clean([string]$from, [string]$to) {
    $skipDirs = @("shots", "node_modules", "dist")
    $skipFiles = @("*.flag", "run.out.txt", "run.err.txt", "*.log", "imgui.ini", ".moonbasic_run.mb")
    $base = (Resolve-Path $from).Path
    Get-ChildItem $base -Recurse -File | ForEach-Object {
        $rel = $_.FullName.Substring($base.Length + 1)
        $parts = $rel.Split([IO.Path]::DirectorySeparatorChar)
        if ($parts.Count -gt 1 -and ($parts[0..($parts.Count - 2)] | Where-Object { $skipDirs -contains $_ })) { return }
        foreach ($pat in $skipFiles) { if ($_.Name -like $pat) { return } }
        $dest = Join-Path $to $rel
        New-Item -ItemType Directory -Force -Path (Split-Path $dest) | Out-Null
        Copy-Item $_.FullName $dest
    }
}

New-Item -ItemType Directory -Force -Path $distRoot | Out-Null
Reset-Folder $sdk

Write-Host "1. Compiler and language server (stripped)..."
$env:CGO_ENABLED = "1"
Invoke-Checked "go build moon" { go build -trimpath -ldflags "-s -w" -o (Join-Path $sdk "moon.exe") ./cmd/moon }
Invoke-Checked "go build moonsls" { go build -trimpath -ldflags "-s -w" -o (Join-Path $sdk "moonsls.exe") ./cmd/moonsls }
Copy-Item "third_party\windows\*.dll" $sdk

Write-Host "2. IDE..."
$neu = Join-Path $root "ide\bin\neutralino-win_x64.exe"
if (-not (Test-Path $neu)) { throw "ide\bin\neutralino-win_x64.exe is missing; run .\scripts\build_ide.ps1 once to fetch it" }
Push-Location "ide\frontend"
try {
    if (-not (Test-Path "node_modules")) { Invoke-Checked "npm ci" { npm ci } }
    Invoke-Checked "npm run build" { npm run build }
} finally { Pop-Location }
Push-Location "ide"
try {
    # A GUI program: no console window behind the IDE
    Invoke-Checked "go build ide" { go build -trimpath -ldflags "-s -w -H windowsgui" -o (Join-Path $sdk "moonbasic.exe") . }
} finally { Pop-Location }
Copy-Item $neu $sdk
Copy-Item "ide\neutralino.config.json" $sdk

Write-Host "3. Docs, examples, games, editor support..."
Copy-Clean "docs" (Join-Path $sdk "docs")
Copy-Clean "examples" (Join-Path $sdk "examples")
Copy-Clean "games" (Join-Path $sdk "games")
Copy-Clean "editors" (Join-Path $sdk "editors")
Copy-Item README.md, LICENSE $sdk
Copy-Item "scripts\SDK_README.txt" (Join-Path $sdk "START_HERE.txt")

Write-Host "4. Standalone games, packaged by the SDK's own moon.exe (no Go)..."
$games = @(
    @{ Dir = "games\hunter"; Name = "Greenwood" },
    @{ Dir = "games\black_veil"; Name = "TheBlackVeil" }
)
$outputs = @($sdk)
foreach ($g in $games) {
    $out = Join-Path $distRoot "$($g.Name)-$version-windows"
    Reset-Folder $out
    $env:MOON_BUILD_PREBUILT = "1"
    try {
        Invoke-Checked "moon build $($g.Name)" { & (Join-Path $sdk "moon.exe") build (Join-Path $g.Dir "main.mb") -o $out -profile release }
    } finally { Remove-Item Env:MOON_BUILD_PREBUILT }
    $outputs += $out
}

if (-not $NoZip) {
    Write-Host "5. Zips..."
    foreach ($o in $outputs) {
        $zip = "$o.zip"
        if (Test-Path $zip) { Remove-Item $zip -Force }
        Compress-Archive -Path $o -DestinationPath $zip -CompressionLevel Optimal
    }
}

Write-Host ""
Write-Host "Done:"
foreach ($o in $outputs) {
    $mb = [math]::Round(((Get-ChildItem $o -Recurse -File | Measure-Object Length -Sum).Sum) / 1MB, 1)
    Write-Host ("  {0}  ({1} MB)" -f $o, $mb)
}
