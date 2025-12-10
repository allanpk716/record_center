# Test different detail indexes
Write-Host "=== Testing Detail Indexes ===" -ForegroundColor Green

$shell = New-Object -ComObject Shell.Application
$portable = $shell.NameSpace(17)

if ($portable) {
    $device = $portable.Items() | Where-Object { $_.Name -like "*SR302*" } | Select-Object -First 1

    if ($device) {
        try {
            $deviceFolder = $device.GetFolder()

            # Find an .opus file
            $opusFile = $null
            foreach ($item in $deviceFolder.Items()) {
                if ($item.Name -like "*.opus") {
                    $opusFile = $item
                    break
                }
            }

            if ($opusFile) {
                Write-Host "Testing with file: $($opusFile.Name)" -ForegroundColor Cyan
                Write-Host ""

                # Test different detail indexes (0-20)
                for ($i = 0; $i -le 20; $i++) {
                    $detail = $deviceFolder.GetDetailsOf($opusFile, $i)
                    Write-Host "Index $i : '$detail'" -ForegroundColor White
                }
            } else {
                Write-Host "No .opus file found" -ForegroundColor Red
            }

        } catch {
            Write-Host "Error: $($_.Exception.Message)" -ForegroundColor Red
        }
    } else {
        Write-Host "Device not found" -ForegroundColor Red
    }
} else {
    Write-Host "Cannot get portable namespace" -ForegroundColor Red
}

Write-Host "Test completed" -ForegroundColor Green