# Simple test for MTP device file size access
Write-Host "=== Testing MTP File Size Access ===" -ForegroundColor Green

# Method 1: Shell Application
Write-Host "`nMethod 1: Shell.Application" -ForegroundColor Cyan

try {
    $shell = New-Object -ComObject Shell.Application
    $portable = $shell.NameSpace(17)

    if ($portable) {
        $device = $portable.Items() | Where-Object { $_.Name -eq "SR302" } | Select-Object -First 1

        if ($device) {
            Write-Host "Found SR302 device" -ForegroundColor Green
            $deviceFolder = $device.GetFolder()

            # Search for .opus files
            function Find-OpusFiles($folder, $depth = 0) {
                if ($depth -gt 5) { return }

                foreach ($item in $folder.Items()) {
                    if ($item.IsFolder) {
                        try {
                            $subFolder = $item.GetFolder()
                            Find-OpusFiles $subFolder ($depth + 1)
                        } catch {
                            continue
                        }
                    } elseif ($item.Name -like "*.opus") {
                        Write-Host "File: $($item.Name)" -ForegroundColor White

                        # Try different methods to get file size
                        $size = 0
                        $method = "Unknown"

                        if ($item.Size -gt 0) {
                            $size = $item.Size
                            $method = "Size property"
                        } else {
                            try {
                                $details = $folder.GetDetailsOf($item, 1)
                                if ($details -match '(\d+(?:,\d+)*)\s*(KB|MB|GB|B)') {
                                    $num = $matches[1] -replace ',', ''
                                    $unit = $matches[2]
                                    $size = switch ($unit) {
                                        "KB" { [long][double]$num * 1KB }
                                        "MB" { [long][double]$num * 1MB }
                                        "GB" { [long][double]$num * 1GB }
                                        "B"  { [long][double]$num }
                                    }
                                    $method = "GetDetailsOf"
                                }
                            } catch {
                                $method = "Failed"
                            }
                        }

                        $sizeMB = [math]::Round($size/1MB, 2)
                        Write-Host "  Size: $size bytes ($sizeMB MB) - Method: $method" -ForegroundColor Cyan
                    }
                }
            }

            Find-OpusFiles $deviceFolder

        } else {
            Write-Host "SR302 device not found" -ForegroundColor Red
        }
    } else {
        Write-Host "Cannot access portable devices" -ForegroundColor Red
    }
} catch {
    Write-Host "Shell COM failed: $($_.Exception.Message)" -ForegroundColor Red
}

# Method 2: Direct path access
Write-Host "`nMethod 2: Direct path access" -ForegroundColor Cyan

$testPaths = @(
    "\\?\usb#vid_2207&pid_0011*",
    "\\localhost\c$\Users",
    "C:\Users"
)

foreach ($path in $testPaths) {
    Write-Host "Testing path: $path" -ForegroundColor Yellow
    try {
        if (Test-Path $path) {
            $items = Get-ChildItem $path -ErrorAction Stop | Select-Object -First 3
            Write-Host "  Accessible, found $($items.Count) items" -ForegroundColor Green
        } else {
            Write-Host "  Path not accessible" -ForegroundColor Red
        }
    } catch {
        Write-Host "  Error: $($_.Exception.Message)" -ForegroundColor Red
    }
}

Write-Host "`n=== Test Complete ===" -ForegroundColor Green