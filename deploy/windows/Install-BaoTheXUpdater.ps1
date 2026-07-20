[CmdletBinding()]
param(
  [string]$RepositoryUrl = "https://github.com/madebyduy/BaoTheX.git",
  [string]$GitHubRepository = "madebyduy/BaoTheX",
  [string]$Branch = "main",
  [string]$InstallRoot = (Join-Path $env:LOCALAPPDATA "BaoTheX"),
  [string]$EnvironmentFile = (Join-Path (Resolve-Path "$PSScriptRoot\..\..").Path ".env"),
  [string]$FrontendOrigin = "https://baothex-web.universeapd.workers.dev",
  [string]$MediaPublicBaseUrl = "https://provision-materials-cameron-pure.trycloudflare.com",
  [switch]$EnableWorker,
  [int]$IntervalMinutes = 2
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

if ($IntervalMinutes -lt 1) { throw "IntervalMinutes must be at least 1." }
$resolvedEnv = (Resolve-Path -LiteralPath $EnvironmentFile).Path
$resolvedInstall = [IO.Path]::GetFullPath($InstallRoot)
$opsRoot = Join-Path $resolvedInstall "ops"
$configRoot = Join-Path $resolvedInstall "config"
$configPath = Join-Path $configRoot "deployment.json"
$installedUpdater = Join-Path $opsRoot "Sync-BaoTheX.ps1"

New-Item -ItemType Directory -Force -Path $resolvedInstall, $opsRoot, $configRoot | Out-Null
Copy-Item -LiteralPath (Join-Path $PSScriptRoot "Sync-BaoTheX.ps1") -Destination $installedUpdater -Force

[pscustomobject]@{
  repository_url       = $RepositoryUrl
  github_repository    = $GitHubRepository
  branch               = $Branch
  install_root         = $resolvedInstall
  environment_file     = $resolvedEnv
  frontend_origin      = $FrontendOrigin.TrimEnd("/")
  media_public_base_url = $MediaPublicBaseUrl.TrimEnd("/")
  health_url           = "http://127.0.0.1:8081/healthz"
  enable_worker        = $EnableWorker.IsPresent
} | ConvertTo-Json | Set-Content -LiteralPath $configPath -Encoding utf8

$taskName = "BaoTheX Pull Deploy"
$powerShell = (Get-Command powershell.exe).Source
$arguments = "-NoProfile -ExecutionPolicy Bypass -File `"$installedUpdater`" -ConfigPath `"$configPath`""
$action = New-ScheduledTaskAction -Execute $powerShell -Argument $arguments
$trigger = New-ScheduledTaskTrigger -Once -At (Get-Date).AddMinutes(1) `
  -RepetitionInterval (New-TimeSpan -Minutes $IntervalMinutes)
$settings = New-ScheduledTaskSettingsSet -MultipleInstances IgnoreNew -StartWhenAvailable `
  -ExecutionTimeLimit (New-TimeSpan -Minutes 20)
$principal = New-ScheduledTaskPrincipal -UserId ([Security.Principal.WindowsIdentity]::GetCurrent().Name) `
  -LogonType Interactive -RunLevel Limited

Register-ScheduledTask -TaskName $taskName -Action $action -Trigger $trigger `
  -Settings $settings -Principal $principal -Description "Pull and safely deploy BaoTheX main" `
  -Force | Out-Null

Write-Output "Installed scheduled task: $taskName"
Write-Output "Runtime: $resolvedInstall"
Write-Output "Config:  $configPath"
Write-Output "Worker enabled: $($EnableWorker.IsPresent)"
Write-Output "Run now with: Start-ScheduledTask -TaskName '$taskName'"
