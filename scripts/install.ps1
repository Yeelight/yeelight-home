$ErrorActionPreference = "Stop"

$Repo = if ($env:YEELIGHT_HOME_REPO) { $env:YEELIGHT_HOME_REPO } else { "yeelight/yeelight-home" }
$Version = if ($env:YEELIGHT_HOME_VERSION) { $env:YEELIGHT_HOME_VERSION } else { "latest" }
$InstallDir = if ($env:YEELIGHT_HOME_INSTALL_DIR) { $env:YEELIGHT_HOME_INSTALL_DIR } else { Join-Path $env:LOCALAPPDATA "Programs\YeelightHome\bin" }
$BinName = "yeelight-home.exe"

function Get-Arch {
  switch ($env:PROCESSOR_ARCHITECTURE) {
    "AMD64" { "amd64"; break }
    "ARM64" { "arm64"; break }
    default { throw "unsupported architecture: $env:PROCESSOR_ARCHITECTURE" }
  }
}

$Arch = Get-Arch
$Asset = "yeelight-home-windows-$Arch.zip"
$BaseUrl = "https://github.com/$Repo/releases"
if ($Version -eq "latest") {
  $Url = "$BaseUrl/latest/download/$Asset"
} else {
  $Url = "$BaseUrl/download/$Version/$Asset"
}

$TempDir = Join-Path ([System.IO.Path]::GetTempPath()) ("yeelight-home-install-" + [System.Guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Path $TempDir | Out-Null

try {
  $Archive = Join-Path $TempDir $Asset
  Invoke-WebRequest -Uri $Url -OutFile $Archive
  $ChecksumsUrl = if ($Version -eq "latest") { "$BaseUrl/latest/download/checksums.txt" } else { "$BaseUrl/download/$Version/checksums.txt" }
  $ChecksumsPath = Join-Path $TempDir "checksums.txt"
  Invoke-WebRequest -Uri $ChecksumsUrl -OutFile $ChecksumsPath
  $Expected = (Get-Content $ChecksumsPath | Where-Object { $_ -match "\s$([regex]::Escape($Asset))$" } | ForEach-Object { ($_ -split "\s+")[0] } | Select-Object -First 1)
  $Actual = (Get-FileHash -Algorithm SHA256 -Path $Archive).Hash.ToLowerInvariant()
  if ([string]::IsNullOrWhiteSpace($Expected) -or $Expected.ToLowerInvariant() -ne $Actual) {
    throw "checksum verification failed for $Asset"
  }
  Expand-Archive -Path $Archive -DestinationPath $TempDir -Force
  $Binary = Join-Path $TempDir $BinName
  if (!(Test-Path $Binary)) {
    throw "release archive does not contain $BinName"
  }

  New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
  $Target = Join-Path $InstallDir $BinName
  Copy-Item -Path $Binary -Destination $Target -Force

  & $Target version
  Write-Output "installed yeelight-home to $Target"
  $UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
  if (($UserPath -split ";") -notcontains $InstallDir) {
    $UpdatedPath = if ([string]::IsNullOrWhiteSpace($UserPath)) { $InstallDir } else { "$UserPath;$InstallDir" }
    [Environment]::SetEnvironmentVariable("Path", $UpdatedPath, "User")
    Write-Output "added $InstallDir to the user PATH. Open a new terminal before running yeelight-home."
  }
} finally {
  Remove-Item -Recurse -Force $TempDir -ErrorAction SilentlyContinue
}
