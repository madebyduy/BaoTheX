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
$env:NEXT_PUBLIC_API_URL = "http://localhost:8081"
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
Start-DevWindow "BaoTheX Frontend" (Join-Path $root "apps\web") "npm run dev"

Write-Host "Đã khởi động frontend, API và worker." -ForegroundColor Green
Write-Host "Frontend: http://localhost:3000"
Write-Host "API:      http://localhost:8081"
