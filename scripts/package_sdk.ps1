# Release packaging: the moonBASIC SDK and the sample games, ready to zip and send.
#
#   .\scripts\package_sdk.ps1            # everything, into dist\
#   .\scripts\package_sdk.ps1 -NoZip     # folders only
#
# The SDK folder a user unzips:
#
#   moonBASIC.exe        the IDE: the one thing to open
#   Start Here.html      docs\GETTING_STARTED.md as a page
#   LICENSE.txt
#   docs\                every docs\*.md as a page, and an index
#   examples\  games\    programs and games with source
#   extras\vscode\       VS Code extension
#   bin\                 moon.exe, moonsls.exe, the DLLs, the IDE's window (Neutralino)
#
# and each game in games\ as a standalone release build with its HOW_TO_PLAY.txt,
# plus a .zip of each folder.
#
# Needs Go with a C compiler (MinGW-w64 gcc), Node.js for the IDE frontend,
# Python with the markdown package for the pages, and ide\bin\neutralino-win_x64.exe
# (.\scripts\build_ide.ps1 fetches it). The SDK itself needs none of these.

param([switch]$NoZip)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

$version = (Select-String -Path "cmd\moon\main.go" -Pattern 'const version = "([^"]+)"').Matches[0].Groups[1].Value
$distRoot = Join-Path $root "dist"
$sdk = Join-Path $distRoot "moonBASIC-$version-windows-amd64"
$bin = Join-Path $sdk "bin"

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
    Get-ChildItem $base -Recurse -File -Force | ForEach-Object {
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
New-Item -ItemType Directory -Force -Path $bin | Out-Null

Write-Host "1. Compiler and language server (stripped), into bin\..."
$env:CGO_ENABLED = "1"
Invoke-Checked "go build moon" { go build -trimpath -ldflags "-s -w" -o (Join-Path $bin "moon.exe") ./cmd/moon }
Invoke-Checked "go build moonsls" { go build -trimpath -ldflags "-s -w" -o (Join-Path $bin "moonsls.exe") ./cmd/moonsls }
Copy-Item "third_party\windows\*.dll" $bin

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
    Invoke-Checked "go build ide" { go build -trimpath -ldflags "-s -w -H windowsgui" -o (Join-Path $sdk "moonBASIC.exe") . }
} finally { Pop-Location }
Copy-Item $neu $bin
Copy-Item "ide\neutralino.config.json" $bin

Write-Host "3. Pages, examples, games, extras..."
Invoke-Checked "docs_to_html" { python scripts\docs_to_html.py docs $sdk }
Copy-Clean "examples" (Join-Path $sdk "examples")
Copy-Clean "games" (Join-Path $sdk "games")
Copy-Clean "editors\vscode" (Join-Path $sdk "extras\vscode")
Copy-Item LICENSE (Join-Path $sdk "LICENSE.txt")

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
        Invoke-Checked "moon build $($g.Name)" { & (Join-Path $bin "moon.exe") build (Join-Path $g.Dir "main.mb") -o $out -profile release }
    } finally { Remove-Item Env:MOON_BUILD_PREBUILT }
    $howTo = Join-Path $g.Dir "HOW_TO_PLAY.txt"
    if (Test-Path $howTo) { Copy-Item $howTo $out }
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
