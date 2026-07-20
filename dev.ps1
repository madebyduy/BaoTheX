$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $MyInvocation.MyCommand.Path

if (-not (Test-Path (Join-Path $root ".env"))) {
  throw "Chưa có file .env trong thư mục dự án."
}

# Nạp .env vào process hiện tại; không in ra các secret.
Get-Content (Join-Path $root ".env") | ForEach-Object {
  if ($_ -match '^\s*([^#=][^=]*)=(.*)$') {
    $name = $matches[1].Trim()
    $value = $matches[2].Trim()
    [Environment]::SetEnvironmentVariable($name, $value, "Process")
  }
}

$env:API_ADDR = ":8081"
$ngrokURL = "https://miranda-gasmetophytic-unboyishly.ngrok-free.dev"
$env:NEXT_PUBLIC_API_URL = $ngrokURL
$env:GO111MODULE = "on"

function Start-DevWindow([string]$title, [string]$workdir, [string]$command) {
  $safeTitle = $title.Replace('"', '')
  $safeDir = $workdir.Replace('"', '""')
  $safeCommand = $command.Replace('"', '""')
  Start-Process powershell.exe -ArgumentList @(
    "-NoExit",
    "-ExecutionPolicy", "Bypass",
    "-Command", "`$Host.UI.RawUI.WindowTitle='$safeTitle'; Set-Location `"$safeDir`"; $safeCommand"
  )
}

Start-DevWindow "BaoTheX API" $root "go run .\apps\api"
Start-DevWindow "BaoTheX Worker" $root "go run .\apps\worker"

# Ngrok free accounts have one assigned dev domain. Reusing the account's
# endpoint keeps the public API URL stable across agent and project restarts.
$activeNgrok = $null
try {
  $activeNgrok = (Invoke-RestMethod -Uri "http://127.0.0.1:4040/api/tunnels" -TimeoutSec 2).tunnels |
    Where-Object { $_.public_url -eq $ngrokURL -and $_.config.addr -match ':8081/?$' } |
    Select-Object -First 1
} catch {
  # No local ngrok agent is running yet.
}

if (-not $activeNgrok) {
  $ngrokCommand = Get-Command ngrok -ErrorAction SilentlyContinue
  $ngrokExe = if ($ngrokCommand) {
    $ngrokCommand.Source
  } else {
    Join-Path $env:USERPROFILE "Downloads\ngrok-v3-stable-windows-amd64\ngrok.exe"
  }
  if (-not (Test-Path $ngrokExe)) {
    throw "Khong tim thay ngrok.exe. Cai ngrok hoac them ngrok vao PATH."
  }
  Start-DevWindow "BaoTheX Tunnel" $root "& '$ngrokExe' http 8081"
}

Start-DevWindow "BaoTheX Frontend" (Join-Path $root "apps\web") "npm run dev"

Write-Host "Đã khởi động frontend, API và worker." -ForegroundColor Green
Write-Host "Frontend: http://localhost:3000"
Write-Host "API:      http://localhost:8081"
Write-Host "Tunnel:   $ngrokURL"
