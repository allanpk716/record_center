# Test actual file size by copying
Write-Host "=== Testing Real File Size by Copying ===" -ForegroundColor Green

try {
    $shell = New-Object -ComObject Shell.Application
    $portable = $shell.NameSpace(17)

    if ($portable) {
        $device = $portable.Items() | Where-Object { $_.Name -eq "SR302" } | Select-Object -First 1

        if ($device) {
            Write-Host "Found SR302 device" -ForegroundColor Green
            $deviceFolder = $device.GetFolder()

            # Find the .opus file
            function Find-OpusFile($folder, $depth = 0) {
                if ($depth -gt 5) { return $null }

                foreach ($item in $folder.Items()) {
                    if ($item.IsFolder) {
                        try {
                            $subFolder = $item.GetFolder()
                            $result = Find-OpusFile $subFolder ($depth + 1)
                            if ($result) { return $result }
                        } catch {
                            continue
                        }
                    } elseif ($item.Name -like "*.opus") {
                        return $item
                    }
                }
                return $null
            }

            $opusFile = Find-OpusFile $deviceFolder

            if ($opusFile) {
                Write-Host "Found opus file: $($opusFile.Name)" -ForegroundColor White
                Write-Host "Shell Size property: $($opusFile.Size) bytes" -ForegroundColor Yellow

                # Method 1: Copy to temp to get real size
                $tempPath = "$env:TEMP\test_real_size.opus"
                Write-Host "Copying to temp file to get real size..." -ForegroundColor Cyan

                try {
                    $shell.NameSpace($env:TEMP).CopyHere($opusFile, 0)

                    # Wait for copy to complete
                    Start-Sleep -Seconds 2

                    if (Test-Path $tempPath) {
                        $tempFile = Get-Item $tempPath
                        $realSize = $tempFile.Length
                        $realSizeMB = [math]::Round($realSize/1MB, 2)

                        Write-Host "REAL FILE SIZE: $realSize bytes ($realSizeMB MB)" -ForegroundColor Green
                        Write-Host "Shell Size was WRONG: $($opusFile.Size) bytes" -ForegroundColor Red

                        # Clean up
                        Remove-Item $tempPath -Force -ErrorAction SilentlyContinue
                    } else {
                        Write-Host "Copy failed - temp file not found" -ForegroundColor Red
                    }
                } catch {
                    Write-Host "Copy failed: $($_.Exception.Message)" -ForegroundColor Red
                }

                # Method 2: Try using file system path directly
                Write-Host "`nTrying direct file system access..." -ForegroundColor Cyan
                try {
                    # Get the full path from the shell item
                    $fullPath = $opusFile.Path
                    Write-Host "Shell Path: $fullPath" -ForegroundColor Yellow

                    # Try to access via file system
                    if (Test-Path $fullPath) {
                        $fsFile = Get-Item $fullPath
                        Write-Host "File System Size: $($fsFile.Length) bytes" -ForegroundColor Green
                    } else {
                        Write-Host "Cannot access via file system" -ForegroundColor Red
                    }
                } catch {
                    Write-Host "File system access failed: $($_.Exception.Message)" -ForegroundColor Red
                }

            } else {
                Write-Host "No .opus files found" -ForegroundColor Red
            }

        } else {
            Write-Host "SR302 device not found" -ForegroundColor Red
        }
    } else {
        Write-Host "Cannot access portable devices" -ForegroundColor Red
    }
} catch {
    Write-Host "Test failed: $($_.Exception.Message)" -ForegroundColor Red
}

Write-Host "`n=== Test Complete ===" -ForegroundColor Green