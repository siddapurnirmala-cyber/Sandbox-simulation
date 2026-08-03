Write-Host "=============================================" -ForegroundColor Cyan
Write-Host "   SLA LATENCY & ROUTING VISUALIZER (CLI)" -ForegroundColor Cyan
Write-Host "=============================================" -ForegroundColor Cyan
Write-Host "Press Ctrl+C to stop. Target: http://localhost:8081/sandbox" -ForegroundColor Yellow
Write-Host ""

while ($true) {
    $start = Get-Date
    try {
        $response = Invoke-WebRequest -Uri "http://localhost:8081/sandbox" -UseBasicParsing -TimeoutSec 15
        $end = Get-Date
        $duration = ($end - $start).TotalSeconds
        
        $timestamp = (Get-Date).ToString("HH:mm:ss")
        if ($duration -ge 5.0) {
            Write-Host "[$timestamp] HTTP Status: $($response.StatusCode) | Latency: $($duration.ToString('0.000'))s | SLA BREACH! 🚨 (Email Alert Sent)" -ForegroundColor Red
        } else {
            Write-Host "[$timestamp] HTTP Status: $($response.StatusCode) | Latency: $($duration.ToString('0.000'))s | OK ✅" -ForegroundColor Green
        }
    } catch {
        $timestamp = (Get-Date).ToString("HH:mm:ss")
        $end = Get-Date
        $duration = ($end - $start).TotalSeconds
        Write-Host "[$timestamp] Request Failed! Latency: $($duration.ToString('0.000'))s | Error: $_" -ForegroundColor DarkRed
    }
    Start-Sleep -Seconds 1
}
