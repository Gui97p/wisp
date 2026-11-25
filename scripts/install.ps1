$ErrorActionPreference = "Stop"

$Repo = "Gui97p/wisp"
$Base = if ($env:WISP_UPDATE_URL) { $env:WISP_UPDATE_URL } else { "https://github.com/$Repo" }
$Target = if ($env:WISP_HOME) { $env:WISP_HOME } else { Join-Path $env:LOCALAPPDATA "wisp" }

if (-not [Environment]::Is64BitOperatingSystem) {
    throw "install: there is no release for 32-bit Windows"
}

$Target = $Target.TrimEnd('\', '/')
$systemFolders = @($env:SystemRoot, $env:ProgramFiles, ${env:ProgramFiles(x86)}, $env:USERPROFILE, $env:LOCALAPPDATA, (Split-Path $env:SystemRoot), $env:SystemDrive + "\") | Where-Object { $_ }
if (($Target -eq "") -or ($systemFolders | Where-Object { $_.TrimEnd('\', '/') -ieq $Target })) {
    throw "install: WISP_HOME=$Target would replace a system folder: choose a folder of its own"
}
if ((Test-Path $Target) -and -not ((Test-Path (Join-Path $Target "bin\wisp.exe")) -and (Test-Path (Join-Path $Target "std")))) {
    throw "install: $Target exists and is not a Wisp installation: choose a folder that does not exist, or one that already holds Wisp"
}

$tag = $env:WISP_VERSION
if (-not $tag) {
    try {
        $tag = (Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest" -Headers @{ "User-Agent" = "wisp-install" }).tag_name
    } catch {
        throw "install: no release has been published yet"
    }
}
if ($tag -notmatch '^v\d+\.\d+\.\d+$') {
    throw "install: no release has been published yet"
}
$version = $tag.Substring(1)
$name = "wisp-$version-windows-amd64.zip"

$work = Join-Path ([IO.Path]::GetTempPath()) ("wisp-install-" + [Guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Path $work | Out-Null
try {
    Write-Host "downloading wisp $version"
    $ProgressPreference = "SilentlyContinue"
    Invoke-WebRequest -Uri "$Base/releases/download/$tag/$name" -OutFile (Join-Path $work $name)
    Invoke-WebRequest -Uri "$Base/releases/download/$tag/SHA256SUMS" -OutFile (Join-Path $work "SHA256SUMS")

    $line = Get-Content (Join-Path $work "SHA256SUMS") | Where-Object { $_ -match [regex]::Escape($name) + '$' } | Select-Object -First 1
    if (-not $line) { throw "install: SHA256SUMS has no checksum for $name" }
    $want = ($line -split '\s+')[0].ToLower()
    $got = (Get-FileHash -Algorithm SHA256 (Join-Path $work $name)).Hash.ToLower()
    if ($got -ne $want) { throw "install: the checksum of $name does not match" }

    Expand-Archive -Path (Join-Path $work $name) -DestinationPath (Join-Path $work "out")
    if (Test-Path $Target) { Remove-Item -Recurse -Force $Target }
    New-Item -ItemType Directory -Force -Path (Split-Path $Target) | Out-Null
    Move-Item (Join-Path $work "out\wisp") $Target
} finally {
    Remove-Item -Recurse -Force $work -ErrorAction SilentlyContinue
}

Write-Host "wisp $version installed in $Target"

$bin = Join-Path $Target "bin"
$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if (($userPath -split ';') -notcontains $bin) {
    [Environment]::SetEnvironmentVariable("Path", ($userPath.TrimEnd(';') + ";" + $bin), "User")
    Write-Host "added $bin to your PATH: open a new terminal to use it"
}

& (Join-Path $bin "wisp.exe") doctor
