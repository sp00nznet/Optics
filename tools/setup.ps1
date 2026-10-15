# Setup.cmd's body. Runs the same commands as README "Step by step", in order,
# and stops with one sentence on what to do next if any of them fails. Details
# go to setup.log in the repo folder.
$ErrorActionPreference = 'Stop'
$root = Split-Path $PSScriptRoot -Parent
$log = Join-Path $root 'setup.log'
Set-Location $root
"--- setup $(Get-Date -Format s)" | Out-File $log -Append -Encoding utf8

function Step($what) { Write-Host "`n== $what" -ForegroundColor Cyan; "== $what" | Out-File $log -Append -Encoding utf8 }
function Fail($what) { Write-Host "`n$what" -ForegroundColor Red; $what | Out-File $log -Append -Encoding utf8; exit 1 }
function Run($exe, [string[]]$argv) {
    & $exe @argv 2>&1 | Tee-Object -FilePath $log -Append | Out-Host
    if ($LASTEXITCODE -ne 0) { throw "$exe exited with $LASTEXITCODE" }
}

# 1. The MSVC C compiler (Visual Studio 2022 or its Build Tools) and CMake.
Step 'Checking for the Visual Studio C++ build tools'
$vswhere = "${env:ProgramFiles(x86)}\Microsoft Visual Studio\Installer\vswhere.exe"
function FindVS { if (Test-Path $vswhere) { & $vswhere -latest -products * -requires Microsoft.VisualStudio.Component.VC.Tools.x86.x64 -property installationPath } }
$vs = FindVS
if (-not $vs) {
    Write-Host 'The Visual Studio 2022 C++ Build Tools are not installed. Optics needs them to compile.'
    Write-Host 'Installing them with winget downloads about 3 GB and takes 10-20 minutes.'
    if ((Read-Host 'Install them now? [y/N]') -notmatch '^[yY]') { Fail 'Install "Build Tools for Visual Studio 2022" with the "Desktop development with C++" workload, then run Setup.cmd again.' }
    if (-not (Get-Command winget -ErrorAction SilentlyContinue)) { Fail 'winget is missing: install "App Installer" from the Microsoft Store, or install the Build Tools by hand, then rerun.' }
    Run winget @('install', '--id', 'Microsoft.VisualStudio.2022.BuildTools', '-e', '--accept-package-agreements', '--accept-source-agreements',
                 '--override', '--quiet --wait --norestart --add Microsoft.VisualStudio.Workload.VCTools --includeRecommended')
    $vs = FindVS
    if (-not $vs) { Fail 'The Build Tools install did not finish. Restart Windows and run Setup.cmd again.' }
}
Write-Host "found: $vs"

Step 'Checking for CMake'
$cmake = (Get-Command cmake -ErrorAction SilentlyContinue).Source
if (-not $cmake) { $cmake = Join-Path $vs 'Common7\IDE\CommonExtensions\Microsoft\CMake\CMake\bin\cmake.exe' }
if (-not (Test-Path $cmake)) { Fail 'CMake was not found. Add "C++ CMake tools for Windows" in the Visual Studio Installer, or install CMake from cmake.org, then rerun.' }
Write-Host "found: $cmake"

try {
    # 2-4. Configure, build, check. Reruns are incremental.
    Step 'Configuring';   Run $cmake @('-B', 'build', '-A', 'x64')
    Step 'Building';      Run $cmake @('--build', 'build', '--config', 'Release')
    Step 'Self-test';     Run (Join-Path $root 'build\bin\optics-check.exe') @('--selftest')
} catch {
    Fail "The build failed ($_). The full output is in $log."
}

# 5. A shortcut to run it.
Step 'Making the Optics shortcut'
$lnk = (New-Object -ComObject WScript.Shell).CreateShortcut((Join-Path $root 'Optics.lnk'))
$lnk.TargetPath = Join-Path $root 'build\bin\Optics.exe'
$lnk.WorkingDirectory = $root
$lnk.Save()
Write-Host "`nDone. Double-click Optics in this folder, then Browse to a folder of models." -ForegroundColor Green
