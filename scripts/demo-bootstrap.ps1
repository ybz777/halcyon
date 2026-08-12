#Requires -Version 5.1
<#
.SYNOPSIS
  Seeds the Halcyon local stack with demo identity, organization, service accounts, and sample API keys.

.DESCRIPTION
  Idempotent bootstrap for portfolio demo mode. Uses real auth/org APIs and PostgreSQL for
  service-account records. Writes apps/web/.env.local with VITE_DEMO_* values.

  Prerequisites: docker compose stack running (postgres, auth, organization, gateway).

.EXAMPLE
  .\scripts\demo-bootstrap.ps1
#>
param(
  [string]$GatewayUrl = "http://localhost:8088",
  [string]$OrganizationUrl = "http://localhost:8081",
  [string]$DemoEmail = "demo@halcyon.demo",
  [string]$DemoPassword = "HalcyonDemo!2026",
  [string]$OrgName = "Halcyon Demo"
)

$ErrorActionPreference = "Stop"
$RepoRoot = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
$StatePath = Join-Path $RepoRoot "scripts\.demo-state.json"
$WebEnvPath = Join-Path $RepoRoot "apps\web\.env.local"

$ServiceAccounts = @(
  @{ Id = "svc_demoGatewayBot001"; Name = "API Gateway"; Description = "Edge routing and rate limiting" },
  @{ Id = "svc_demoStagingBot001"; Name = "Staging Integrations"; Description = "Webhook delivery and connectors" },
  @{ Id = "svc_demoWorkflowJob01"; Name = "Workflow Runner"; Description = "Background jobs and workflows" }
)

function Wait-HttpOk([string]$Url, [int]$Retries = 60) {
  for ($i = 0; $i -lt $Retries; $i++) {
    try {
      $r = Invoke-WebRequest -Uri $Url -UseBasicParsing -TimeoutSec 3
      if ($r.StatusCode -ge 200 -and $r.StatusCode -lt 300) { return }
    } catch { }
    Start-Sleep -Seconds 2
  }
  throw "Timed out waiting for $Url"
}

function Invoke-Api {
  param(
    [string]$Method,
    [string]$Path,
    [object]$Body = $null,
    [string]$Token = $null,
    [string]$OrgId = $null,
    [string]$BaseUrl = $GatewayUrl,
    [switch]$Idempotent
  )
  $headers = @{ Accept = "application/json" }
  if ($Token) { $headers.Authorization = "Bearer $Token" }
  if ($OrgId) { $headers["X-Organization-ID"] = $OrgId }
  if ($Idempotent) { $headers["Idempotency-Key"] = [guid]::NewGuid().ToString() }
  $params = @{
    Uri = "$BaseUrl$Path"
    Method = $Method
    Headers = $headers
    ContentType = "application/json"
  }
  if ($null -ne $Body) { $params.Body = ($Body | ConvertTo-Json -Depth 6) }
  try {
    return Invoke-RestMethod @params
  } catch {
    $resp = $_.Exception.Response
    if ($resp) {
      $statusCode = [int]$resp.StatusCode
      $reader = New-Object System.IO.StreamReader($resp.GetResponseStream())
      $detail = $reader.ReadToEnd()
      $reader.Close()
      Write-Host ""
      Write-Host "API request failed:" -ForegroundColor Red
      Write-Host "  $Method $BaseUrl$Path"
      Write-Host "  Status: $statusCode"
      if ($detail) {
        Write-Host "  Response body:"
        Write-Host $detail
      } else {
        Write-Host "  Response body: (empty)"
      }
      Write-Host ""
      throw "API $Method $Path failed ($statusCode): $detail"
    }
    throw
  }
}

Write-Host "Waiting for gateway health..."
Wait-HttpOk "$GatewayUrl/api/v1/health"

# Register (ignore if exists) then login
try {
  Invoke-Api -Method POST -Path "/api/v1/auth/register" -Body @{ email = $DemoEmail; password = $DemoPassword } | Out-Null
  Write-Host "Registered demo user."
} catch {
  Write-Host "Demo user already exists or register skipped."
}

$login = Invoke-Api -Method POST -Path "/api/v1/auth/login" -Body @{ email = $DemoEmail; password = $DemoPassword }
$token = $login.accessToken
if (-not $token) { throw "Login did not return accessToken" }
Write-Host "Logged in as $DemoEmail."

$me = Invoke-Api -Method GET -Path "/api/v1/auth/me" -Token $token
$userId = $me.id
Write-Host "Demo user id: $userId"

# Grant admin role for apikey:manage and full console access
$assignAdmin = @"
INSERT INTO user_roles (user_external_id, role, org_id)
VALUES ('$userId', 'admin', NULL)
ON CONFLICT (user_external_id, role, org_id) DO NOTHING;
"@
docker exec -i halcyon-postgres psql -U halcyon -d halcyon -c $assignAdmin | Out-Null
Write-Host "Ensured admin role for demo user."

# Re-login to refresh JWT permissions
$login = Invoke-Api -Method POST -Path "/api/v1/auth/login" -Body @{ email = $DemoEmail; password = $DemoPassword }
$token = $login.accessToken

$orgId = $null
if (Test-Path $StatePath) {
  $state = Get-Content $StatePath -Raw | ConvertFrom-Json
  if ($state.orgId) { $orgId = $state.orgId }
}

if (-not $orgId) {
  # Gateway does not proxy POST /api/v1/organizations; create via organization service directly.
  $org = Invoke-Api -Method POST -Path "/api/v1/organizations" -Body @{ name = $OrgName } -Token $token -BaseUrl $OrganizationUrl -Idempotent
  $orgId = $org.id
  Write-Host "Created organization '$OrgName' ($orgId)."
} else {
  Write-Host "Reusing organization $orgId from state."
}

# Seed service accounts in auth database
foreach ($sa in $ServiceAccounts) {
  $sql = @"
INSERT INTO service_accounts (id, organization_id, name, description, status)
VALUES ('$($sa.Id)', '$orgId', '$($sa.Name)', '$($sa.Description)', 'active')
ON CONFLICT (id) DO NOTHING;
"@
  docker exec -i halcyon-postgres psql -U halcyon -d halcyon -c $sql | Out-Null
}
Write-Host "Seeded $($ServiceAccounts.Count) service accounts."

# Seed additional demo users as org members (auth users + memberships)
$extraUsers = @(
  @{ Email = "alex.morgan@halcyon.demo"; Role = "admin" },
  @{ Email = "jordan.lee@halcyon.demo"; Role = "member" },
  @{ Email = "sam.patel@halcyon.demo"; Role = "viewer" }
)

foreach ($u in $extraUsers) {
  try {
    Invoke-Api -Method POST -Path "/api/v1/auth/register" -Body @{ email = $u.Email; password = "HalcyonDemo!2026" } | Out-Null
  } catch { }
  try {
    $uLogin = Invoke-Api -Method POST -Path "/api/v1/auth/login" -Body @{ email = $u.Email; password = "HalcyonDemo!2026" }
    $uMe = Invoke-Api -Method GET -Path "/api/v1/auth/me" -Token $uLogin.accessToken
    $memberSql = @"
INSERT INTO memberships (external_id, organization_external_id, user_external_id, role)
VALUES (
  'mem_' || substr(md5(random()::text), 1, 16),
  '$orgId',
  '$($uMe.id)',
  '$($u.Role)'
)
ON CONFLICT (organization_external_id, user_external_id) DO NOTHING;
"@
    docker exec -i halcyon-postgres psql -U halcyon -d halcyon_organization -c $memberSql | Out-Null
  } catch {
    Write-Warning "Could not seed member $($u.Email): $_"
  }
}
Write-Host "Seeded additional organization members."

# Create sample API keys when none exist for the primary service account
$primarySa = $ServiceAccounts[0].Id
$existing = Invoke-Api -Method GET -Path "/api/v1/auth/api-keys?serviceAccountId=$primarySa&page_size=1" -Token $token
if (-not $existing.items -or $existing.items.Count -eq 0) {
  $samples = @(
    @{ name = "CI Deploy Key"; scopes = @("integrations.read"); environment = "test" },
    @{ name = "Production Monitor"; scopes = @("audit.read", "integrations.read"); environment = "live" }
  )
  foreach ($sample in $samples) {
    Invoke-Api -Method POST -Path "/api/v1/auth/api-keys" -Body @{
      name = $sample.name
      serviceAccountId = $primarySa
      organizationId = $orgId
      scopes = $sample.scopes
      environment = $sample.environment
    } -Token $token -Idempotent | Out-Null
  }
  Write-Host "Created sample API keys for $primarySa."
} else {
  Write-Host "API keys already exist for $primarySa - skipping."
}

$stateObj = @{
  orgId = $orgId
  userId = $userId
  primaryServiceAccountId = $primarySa
  bootstrappedAt = (Get-Date).ToUniversalTime().ToString("o")
}
$stateObj | ConvertTo-Json | Set-Content -Path $StatePath -Encoding UTF8

$envContent = @"
# Generated by scripts/demo-bootstrap.ps1 - do not commit.
VITE_DEMO_MODE=true
VITE_DEMO_EMAIL=$DemoEmail
VITE_DEMO_PASSWORD=$DemoPassword
VITE_DEMO_ORG_ID=$orgId
VITE_DEMO_SERVICE_ACCOUNT_ID=$primarySa
"@
Set-Content -Path $WebEnvPath -Value $envContent -Encoding UTF8

Write-Host ""
Write-Host "Demo bootstrap complete."
Write-Host "  Organization: $orgId"
Write-Host "  Service account: $primarySa"
Write-Host "  Web env written: $WebEnvPath"
Write-Host ""
Write-Host "Start the console: cd apps/web; npm run dev"
