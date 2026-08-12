param(
    [string]$Url = "http://localhost:8080/health",
    [int]$Concurrency = 20,
    [int]$Requests = 500
)

$ErrorActionPreference = "Stop"

Write-Host "Load smoke: $Requests requests, concurrency $Concurrency, target $Url"
Write-Host "Environment: local test only — not a production capacity measurement."
Write-Host ""

$jobs = @()
$results = [System.Collections.Concurrent.ConcurrentBag[int]]::new()
$sw = [System.Diagnostics.Stopwatch]::StartNew()

$batch = [Math]::Ceiling($Requests / $Concurrency)
for ($c = 0; $c -lt $Concurrency; $c++) {
    $jobs += Start-Job -ScriptBlock {
        param($url, $count)
        $codes = @()
        for ($i = 0; $i -lt $count; $i++) {
            try {
                $r = Invoke-WebRequest -Uri $url -UseBasicParsing -TimeoutSec 10
                $codes += [int]$r.StatusCode
            } catch {
                if ($_.Exception.Response) {
                    $codes += [int]$_.Exception.Response.StatusCode.value__
                } else {
                    $codes += 0
                }
            }
        }
        return ,$codes
    } -ArgumentList $Url, $batch
}

foreach ($job in $jobs) {
    $codes = Receive-Job -Job $job -Wait
    foreach ($code in $codes) { $results.Add($code) }
    Remove-Job -Job $job
}

$sw.Stop()
$ok = ($results | Where-Object { $_ -ge 200 -and $_ -lt 300 }).Count
$elapsed = $sw.Elapsed.TotalSeconds
$rps = if ($elapsed -gt 0) { [math]::Round($results.Count / $elapsed, 2) } else { 0 }

Write-Host "Completed: $($results.Count) requests in $([math]::Round($elapsed, 2))s"
Write-Host "Throughput: $rps req/s"
Write-Host "2xx responses: $ok / $($results.Count)"
$grouped = $results | Group-Object | Sort-Object Name
foreach ($g in $grouped) {
    Write-Host ("  HTTP {0}: {1}" -f $g.Name, $g.Count)
}
