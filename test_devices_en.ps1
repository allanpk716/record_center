# Check all devices script
Write-Host "=== Device Check ===" -ForegroundColor Green

# 1. Check portable devices
Write-Host "`n1. Checking portable devices:" -ForegroundColor Cyan
try {
    $shell = New-Object -ComObject Shell.Application
    $portable = $shell.NameSpace(17)

    if ($portable) {
        $devices = $portable.Items()
        Write-Host "Found $($devices.Count) portable devices"

        foreach ($device in $devices) {
            Write-Host "  - Device: $($device.Name)" -ForegroundColor White
            Write-Host "    Path: $($device.Path)" -ForegroundColor Gray
        }
    } else {
        Write-Host "Cannot get portable namespace"
    }
} catch {
    Write-Host "Error: $($_.Exception.Message)"
}

# 2. Check WMI USB devices
Write-Host "`n2. WMI USB device check:" -ForegroundColor Cyan
try {
    $usbDevices = Get-WmiObject Win32_PnPEntity | Where-Object {
        $_.DeviceID -like "*USB*" -and
        $_.Name -like "*SR302*"
    }

    if ($usbDevices) {
        foreach ($device in $usbDevices) {
            Write-Host "  USB Device: $($device.Name)" -ForegroundColor Green
            Write-Host "    ID: $($device.DeviceID)" -ForegroundColor Gray
            Write-Host "    Class: $($device.PNPClass)" -ForegroundColor Gray
        }
    } else {
        Write-Host "No SR302 USB devices found"
    }
} catch {
    Write-Host "Error: $($_.Exception.Message)"
}

Write-Host "`nDevice check completed" -ForegroundColor Green