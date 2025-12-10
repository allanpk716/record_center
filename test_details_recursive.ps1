# Test detail indexes recursively
Write-Host "=== Testing Detail Indexes (Recursive) ===" -ForegroundColor Green

$shell = New-Object -ComObject Shell.Application
$portable = $shell.NameSpace(17)

function Find-Any-OpusFile($folder) {
    foreach ($item in $folder.Items()) {
        if ($item.Name -like "*.opus") {
            return $item
        } elseif ($item.IsFolder) {
            try {
                $subFolder = $item.GetFolder()
                $result = Find-Any-OpusFile $subFolder
                if ($result) { return $result }
            } catch {
                # Skip inaccessible folders
            }
        }
    }
    return $null
}

if ($portable) {
    $device = $portable.Items() | Where-Object { $_.Name -like "*SR302*" } | Select-Object -First 1

    if ($device) {
        try {
            $deviceFolder = $device.GetFolder()
            Write-Host "Searching for .opus file..." -ForegroundColor Yellow

            $opusFile = Find-Any-OpusFile $deviceFolder

            if ($opusFile) {
                Write-Host "Found file: $($opusFile.Name)" -ForegroundColor Green
                Write-Host "File path: $($opusFile.Path)" -ForegroundColor Cyan
                Write-Host ""

                # Try to get size directly
                try {
                    $directSize = $opusFile.Size
                    Write-Host "Direct Size property: $directSize" -ForegroundColor White
                } catch {
                    Write-Host "Direct Size property: Error - $($_.Exception.Message)" -ForegroundColor Red
                }

                try {
                    $directLength = $opusFile.Length
                    Write-Host "Direct Length property: $directLength" -ForegroundColor White
                } catch {
                    Write-Host "Direct Length property: Error - $($_.Exception.Message)" -ForegroundColor Red
                }

                # Test GetDetailsOf with different indexes
                Write-Host ""
                Write-Host "Testing GetDetailsOf:" -ForegroundColor Cyan

                # Common indexes that might contain size
                $testIndexes = @(0, 1, 2, 3, 10, 20, 25, 30, 50, 100)
                $folder = $opusFile.ParentFolder

                foreach ($i in $testIndexes) {
                    try {
                        $detail = $folder.GetDetailsOf($opusFile, $i)
                        if ($detail -and $detail.Trim() -ne "") {
                            Write-Host "Index $i : '$detail'" -ForegroundColor White
                        }
                    } catch {
                        # Skip errors
                    }
                }

                # Also test the folder where we found the file
                Write-Host ""
                Write-Host "Testing parent folder GetDetailsOf:" -ForegroundColor Cyan
                foreach ($i in $testIndexes) {
                    try {
                        $detail = $deviceFolder.GetDetailsOf($opusFile, $i)
                        if ($detail -and $detail.Trim() -ne "") {
                            Write-Host "Index $i : '$detail'" -ForegroundColor White
                        }
                    } catch {
                        # Skip errors
                    }
                }

            } else {
                Write-Host "No .opus file found in any subfolder" -ForegroundColor Red
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