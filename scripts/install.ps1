# Install gbi (go-build-it) on Windows from a GitHub release (FR-02b, ADR-012). Windows is beta, like herdr.
#
#   powershell -ExecutionPolicy Bypass -File install.ps1 [-Version YYYY.MM.DD.N] [-Dir $env:LOCALAPPDATA\Programs\gbi]
#
# It downloads the release binary with SHA256SUMS and SHA256SUMS.sig. Then:
#   - It refuses a download whose SHA-256 does not match SHA256SUMS.
#   - It checks the Ed25519 signature of SHA256SUMS with OpenSSL when OpenSSL 3 is on PATH,
#     and refuses on a bad signature. When OpenSSL can't check it, it says so.
#   - It installs gbi.exe into the folder, adds the folder to the user PATH, and checks that it runs.
# The last line is "gbi installed: <path> <version>".
#
# Exit codes: 0 installed, 1 error, 2 verification failed (nothing installed), 3 no release yet.
# For tests: $env:GBI_RELEASE_BASE replaces https://github.com; $env:GBI_PUBLIC_KEY replaces the key.
param(
    [string]$Version = "latest",
    [string]$Dir = (Join-Path $env:LOCALAPPDATA "Programs\gbi")
)
$ErrorActionPreference = "Stop"
$Repo = if ($env:GBI_REPO) { $env:GBI_REPO } else { "techmuch/go-build-it" }
$Base = if ($env:GBI_RELEASE_BASE) { $env:GBI_RELEASE_BASE } else { "https://github.com" }
# The release signing public key. It must match update.PublicKey, README.md and install.sh.
$PublicKey = if ($env:GBI_PUBLIC_KEY) { $env:GBI_PUBLIC_KEY } else { "eJ3eKk81oKaKZlNaLjafw1j1GO+Uueyt51jNMCchih4=" }

function Fail($msg, $code = 1) { [Console]::Error.WriteLine("error: $msg"); exit $code }

$arch = switch ($env:PROCESSOR_ARCHITECTURE) { "ARM64" { "arm64" } default { "amd64" } }
$asset = "gbi_windows_$arch.exe"
$url = if ($Version -eq "latest") { "$Base/$Repo/releases/latest/download" } else { "$Base/$Repo/releases/download/$Version" }
$tmp = Join-Path ([IO.Path]::GetTempPath()) ("gbi-" + [Guid]::NewGuid())
New-Item -ItemType Directory -Path $tmp | Out-Null
try {
    Write-Host "Downloading $asset ($Version) from $Base/$Repo"
    try { Invoke-WebRequest -UseBasicParsing "$url/SHA256SUMS" -OutFile "$tmp\SHA256SUMS" }
    catch {
        Write-Host "No release found at $url."
        Write-Host "If go-build-it has not published a release yet, build it from source: git clone https://github.com/$Repo; cd go-build-it; make build"
        exit 3
    }
    Invoke-WebRequest -UseBasicParsing "$url/SHA256SUMS.sig" -OutFile "$tmp\SHA256SUMS.sig"
    Invoke-WebRequest -UseBasicParsing "$url/$asset" -OutFile "$tmp\$asset"

    # 1. Checksum.
    $want = $null
    foreach ($line in Get-Content "$tmp\SHA256SUMS") {
        $f = $line -split '\s+'
        if ($f.Count -eq 2 -and $f[1].TrimStart('*') -eq $asset) { $want = $f[0] }
    }
    if (-not $want) { Fail "$asset is not listed in SHA256SUMS" 2 }
    $got = (Get-FileHash -Algorithm SHA256 "$tmp\$asset").Hash.ToLower()
    if ($got -ne $want) { Fail "checksum mismatch: the download does not match SHA256SUMS; nothing was installed" 2 }
    Write-Host "Checksum OK (SHA-256 $got)"

    # 2. Signature (Ed25519 over SHA256SUMS), with OpenSSL 3 when available.
    $sigChecked = "no"
    $openssl = Get-Command openssl -ErrorAction SilentlyContinue
    if ($openssl -and ((& openssl pkeyutl -help 2>&1) -join "`n") -match "-rawin") {
        $prefix = [byte[]](0x30,0x2a,0x30,0x05,0x06,0x03,0x2b,0x65,0x70,0x03,0x21,0x00)
        [IO.File]::WriteAllBytes("$tmp\key.der", $prefix + [Convert]::FromBase64String($PublicKey))
        $sigText = (Get-Content -Raw "$tmp\SHA256SUMS.sig").Trim()
        [IO.File]::WriteAllBytes("$tmp\sig.bin", [Convert]::FromBase64String($sigText))
        & openssl pkey -pubin -inform DER -in "$tmp\key.der" -out "$tmp\key.pem" 2>$null
        & openssl pkeyutl -verify -pubin -inkey "$tmp\key.pem" -rawin -in "$tmp\SHA256SUMS" -sigfile "$tmp\sig.bin" *> $null
        if ($LASTEXITCODE -ne 0) { Fail "bad signature: SHA256SUMS was not signed by the go-build-it release key; nothing was installed" 2 }
        $sigChecked = "yes"
        Write-Host "Signature OK (Ed25519, release key)"
    } else {
        Write-Host "Signature NOT checked: OpenSSL 3 is not on PATH. The checksum was checked; gbi checks signatures itself for every later update."
    }

    # 3. Install, put on the user PATH, check that it runs.
    New-Item -ItemType Directory -Force -Path $Dir | Out-Null
    $dest = Join-Path $Dir "gbi.exe"
    Copy-Item "$tmp\$asset" "$dest.new" -Force
    $ver = (& "$dest.new" version).Trim()
    if ($LASTEXITCODE -ne 0) { Remove-Item "$dest.new"; Fail "the downloaded gbi does not run on this machine" }
    if ($Version -ne "latest" -and $ver.TrimStart('v') -ne $Version.TrimStart('v')) { Remove-Item "$dest.new"; Fail "the downloaded gbi reports version $ver, expected $Version" 2 }
    Move-Item -Force "$dest.new" $dest
    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if (-not ($userPath -split ';' | Where-Object { $_ -eq $Dir })) {
        [Environment]::SetEnvironmentVariable("Path", ($userPath.TrimEnd(';') + ";" + $Dir), "User")
        Write-Host "Added $Dir to your user PATH (open a new terminal to use it)."
    }
    Write-Host "Next: gbi doctor --json --fix   (see README: Install with an AI agent)"
    Write-Output "gbi installed: $dest $ver (signature checked: $sigChecked)"
} finally {
    Remove-Item -Recurse -Force $tmp -ErrorAction SilentlyContinue
}
