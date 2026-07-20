[CmdletBinding()]
param(
  [Parameter(Mandatory = $true)]
  [string]$ConfigPath
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

function Write-DeployLog([string]$Message) {
  $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
  Write-Output "[$timestamp] $Message"
}

function Invoke-External([string]$FilePath, [string[]]$Arguments, [string]$WorkingDirectory) {
  Push-Location $WorkingDirectory
  try {
    & $FilePath @Arguments
    if ($LASTEXITCODE -ne 0) {
      throw "$FilePath exited with code $LASTEXITCODE"
    }
  } finally {
    Pop-Location
  }
}

function Import-DotEnv([string]$Path) {
  Get-Content -LiteralPath $Path | ForEach-Object {
    if ($_ -match '^\s*([^#=][^=]*)=(.*)$') {
      [Environment]::SetEnvironmentVariable($matches[1].Trim(), $matches[2].Trim(), "Process")
    }
  }
}

function Test-HttpHealth([string]$Url, [int]$Attempts = 20) {
  for ($attempt = 1; $attempt -le $Attempts; $attempt++) {
    try {
      $response = Invoke-WebRequest -Uri $Url -UseBasicParsing -TimeoutSec 5
      if ($response.StatusCode -eq 200) { return $true }
    } catch {
      # A candidate can take a few seconds to connect to the database.
    }
    Start-Sleep -Seconds 2
  }
  return $false
}

function Test-CiSucceeded([string]$Repository, [string]$Sha) {
  if ($Repository -notmatch '^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$') {
    throw "Invalid GitHub repository name: $Repository"
  }
  $uri = "https://api.github.com/repos/$Repository/actions/workflows/ci.yml/runs?head_sha=$Sha&event=push&per_page=10"
  $headers = @{ Accept = "application/vnd.github+json"; "User-Agent" = "BaoTheX-Pull-Deploy" }
  try {
    $result = Invoke-RestMethod -Uri $uri -Headers $headers -TimeoutSec 15
    return [bool]($result.workflow_runs | Where-Object { $_.conclusion -eq "success" } | Select-Object -First 1)
  } catch {
    Write-DeployLog "Could not verify CI yet: $($_.Exception.Message)"
    return $false
  }
}

function Stop-ManagedProcess([Nullable[int]]$ProcessId, [string]$ReleaseRoot) {
  if (-not $ProcessId.HasValue) { return }
  $process = Get-Process -Id $ProcessId.Value -ErrorAction SilentlyContinue
  if (-not $process) { return }
  $expectedRoot = [IO.Path]::GetFullPath($ReleaseRoot)
  $actualPath = [IO.Path]::GetFullPath($process.Path)
  if (-not $actualPath.StartsWith($expectedRoot, [StringComparison]::OrdinalIgnoreCase)) {
    throw "Refusing to stop PID $($ProcessId.Value): $actualPath is outside $expectedRoot"
  }
  Stop-Process -Id $ProcessId.Value -Force
  $process.WaitForExit(10000)
}

function Start-Release(
  [string]$ReleasePath,
  [string]$LogRoot,
  [bool]$EnableWorker
) {
  $stamp = Get-Date -Format "yyyyMMdd-HHmmss"
  $api = Start-Process -FilePath (Join-Path $ReleasePath "bin\api.exe") `
    -WorkingDirectory $ReleasePath -WindowStyle Hidden -PassThru `
    -RedirectStandardOutput (Join-Path $LogRoot "api-$stamp.out.log") `
    -RedirectStandardError (Join-Path $LogRoot "api-$stamp.err.log")

  $worker = $null
  if ($EnableWorker) {
    $worker = Start-Process -FilePath (Join-Path $ReleasePath "bin\worker.exe") `
      -WorkingDirectory $ReleasePath -WindowStyle Hidden -PassThru `
      -RedirectStandardOutput (Join-Path $LogRoot "worker-$stamp.out.log") `
      -RedirectStandardError (Join-Path $LogRoot "worker-$stamp.err.log")
  }

  return [pscustomobject]@{
    ApiPid    = $api.Id
    WorkerPid = if ($worker) { $worker.Id } else { $null }
  }
}

$resolvedConfig = (Resolve-Path -LiteralPath $ConfigPath).Path
$config = Get-Content -LiteralPath $resolvedConfig -Raw | ConvertFrom-Json
$installRoot = [IO.Path]::GetFullPath([string]$config.install_root)
$sourceRoot = Join-Path $installRoot "source"
$releasesRoot = Join-Path $installRoot "releases"
$stateRoot = Join-Path $installRoot "state"
$logRoot = Join-Path $installRoot "logs"
$statePath = Join-Path $stateRoot "deployment.json"
$envPath = [IO.Path]::GetFullPath([string]$config.environment_file)

New-Item -ItemType Directory -Force -Path $installRoot, $releasesRoot, $stateRoot, $logRoot | Out-Null

$mutex = [Threading.Mutex]::new($false, "BaoTheXLocalDeployment")
if (-not $mutex.WaitOne(0)) {
  Write-DeployLog "Another deployment is already running; skipping."
  exit 0
}

try {
  if (-not (Test-Path -LiteralPath $envPath)) {
    throw "Environment file not found: $envPath"
  }

  if (-not (Test-Path -LiteralPath (Join-Path $sourceRoot ".git"))) {
    Write-DeployLog "Creating isolated deployment clone."
    Invoke-External "git" @("clone", "--filter=blob:none", "--no-checkout", [string]$config.repository_url, $sourceRoot) $installRoot
  }

  Write-DeployLog "Fetching origin/$($config.branch)."
  Invoke-External "git" @("fetch", "--prune", "origin", [string]$config.branch) $sourceRoot
  $targetSha = (& git -C $sourceRoot rev-parse "origin/$($config.branch)").Trim()
  if ($LASTEXITCODE -ne 0 -or $targetSha -notmatch '^[0-9a-f]{40}$') {
    throw "Could not resolve origin/$($config.branch)"
  }

  $previousState = $null
  if (Test-Path -LiteralPath $statePath) {
    $previousState = Get-Content -LiteralPath $statePath -Raw | ConvertFrom-Json
    if ($previousState.sha -eq $targetSha -and (Test-HttpHealth ([string]$config.health_url) 1)) {
      Write-DeployLog "Already running $($targetSha.Substring(0, 12)); no deployment needed."
      exit 0
    }
  }

  if (-not (Test-CiSucceeded ([string]$config.github_repository) $targetSha)) {
    Write-DeployLog "CI is not green for $($targetSha.Substring(0, 12)); deployment deferred."
    exit 0
  }

  $releasePath = Join-Path $releasesRoot $targetSha
  if (-not (Test-Path -LiteralPath $releasePath)) {
    Write-DeployLog "Materializing immutable release $($targetSha.Substring(0, 12))."
    Invoke-External "git" @("worktree", "add", "--detach", $releasePath, $targetSha) $sourceRoot
  }

  Write-DeployLog "Running backend tests."
  Invoke-External "go" @("test", "./...") $releasePath

  New-Item -ItemType Directory -Force -Path (Join-Path $releasePath "bin") | Out-Null
  Write-DeployLog "Building API and worker binaries."
  Invoke-External "go" @("build", "-trimpath", "-o", "bin/api.exe", "./apps/api") $releasePath
  Invoke-External "go" @("build", "-trimpath", "-o", "bin/worker.exe", "./apps/worker") $releasePath

  Import-DotEnv $envPath
  $env:PUBLIC_BASE_URL = [string]$config.frontend_origin
  $env:CORS_ORIGINS = "http://localhost:3000,$($config.frontend_origin)"
  $env:MEDIA_PUBLIC_BASE_URL = [string]$config.media_public_base_url
  $env:TELEGRAM_POLLING = "false"

  Write-DeployLog "Probing the candidate on an isolated port."
  $env:API_ADDR = ":18081"
  $candidateStamp = Get-Date -Format "yyyyMMdd-HHmmss"
  $candidate = Start-Process -FilePath (Join-Path $releasePath "bin\api.exe") `
    -WorkingDirectory $releasePath -WindowStyle Hidden -PassThru `
    -RedirectStandardOutput (Join-Path $logRoot "candidate-$candidateStamp.out.log") `
    -RedirectStandardError (Join-Path $logRoot "candidate-$candidateStamp.err.log")
  try {
    if (-not (Test-HttpHealth "http://127.0.0.1:18081/healthz")) {
      throw "Candidate health check failed; current release was left untouched."
    }
  } finally {
    Stop-Process -Id $candidate.Id -Force -ErrorAction SilentlyContinue
  }

  if ($previousState) {
    Write-DeployLog "Stopping the currently managed release."
    Stop-ManagedProcess ([Nullable[int]]$previousState.api_pid) ([string]$previousState.release_path)
    Stop-ManagedProcess ([Nullable[int]]$previousState.worker_pid) ([string]$previousState.release_path)
  } else {
    $unmanagedListener = Get-NetTCPConnection -LocalPort 8081 -State Listen -ErrorAction SilentlyContinue
    if ($unmanagedListener) {
      throw "Port 8081 is owned by unmanaged PID $($unmanagedListener.OwningProcess). Stop the manual API before the first managed deployment."
    }
  }

  $env:API_ADDR = ":8081"
  Write-DeployLog "Starting release $($targetSha.Substring(0, 12))."
  $started = Start-Release $releasePath $logRoot ([bool]$config.enable_worker)

  $newApi = Get-Process -Id $started.ApiPid -ErrorAction SilentlyContinue
  if (-not $newApi -or -not (Test-HttpHealth ([string]$config.health_url))) {
    Stop-ManagedProcess ([Nullable[int]]$started.ApiPid) $releasePath
    Stop-ManagedProcess ([Nullable[int]]$started.WorkerPid) $releasePath
    if ($previousState) {
      Write-DeployLog "New release failed health-check; rolling back."
      $rollback = Start-Release ([string]$previousState.release_path) $logRoot ([bool]$config.enable_worker)
      $previousState.api_pid = $rollback.ApiPid
      $previousState.worker_pid = $rollback.WorkerPid
      $previousState | ConvertTo-Json | Set-Content -LiteralPath $statePath -Encoding utf8
    }
    throw "New release failed health-check and was rolled back."
  }

  [pscustomobject]@{
    sha          = $targetSha
    release_path = $releasePath
    deployed_at  = (Get-Date).ToUniversalTime().ToString("o")
    api_pid      = $started.ApiPid
    worker_pid   = $started.WorkerPid
  } | ConvertTo-Json | Set-Content -LiteralPath $statePath -Encoding utf8

  Write-DeployLog "Deployment completed: $($targetSha.Substring(0, 12))."
} catch {
  Write-DeployLog "DEPLOYMENT FAILED: $($_.Exception.Message)"
  throw
} finally {
  $mutex.ReleaseMutex()
  $mutex.Dispose()
}
