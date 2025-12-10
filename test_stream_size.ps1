# Test file size using stream reading
Write-Host "=== Testing File Size via Stream Reading ===" -ForegroundColor Green

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

                # Method 1: Try to get file size via stream properties
                Write-Host "`nMethod 1: Stream Properties" -ForegroundColor Cyan
                try {
                    # Get extended properties
                    $extendedProps = $opusFile.ExtendedProperty("System.Size")
                    if ($extendedProps) {
                        Write-Host "System.Size: $extendedProps bytes" -ForegroundColor Green
                    } else {
                        Write-Host "System.Size: Not available" -ForegroundColor Yellow
                    }

                    $fileSize = $opusFile.ExtendedProperty("System.FileSize")
                    if ($fileSize) {
                        Write-Host "System.FileSize: $fileSize bytes" -ForegroundColor Green
                    } else {
                        Write-Host "System.FileSize: Not available" -ForegroundColor Yellow
                    }

                    $length = $opusFile.ExtendedProperty("System.ItemSizeDisplay")
                    if ($length) {
                        Write-Host "System.ItemSizeDisplay: $length" -ForegroundColor Green
                    } else {
                        Write-Host "System.ItemSizeDisplay: Not available" -ForegroundColor Yellow
                    }
                } catch {
                    Write-Host "Extended properties failed: $($_.Exception.Message)" -ForegroundColor Red
                }

                # Method 2: Try to open a stream and read to determine size
                Write-Host "`nMethod 2: Stream Reading" -ForegroundColor Cyan
                try {
                    $tempPath = "$env:TEMP\stream_test.opus"

                    # Try different copy methods
                    $methods = @(
                        @{ Name = "CopyHere"; Flags = 0 },
                        @{ Name = "CopyHere"; Flags = 4 },  # 4 = Don't show progress
                        @{ Name = "CopyHere"; Flags = 16 } # 16 = Yes to all
                    )

                    foreach ($method in $methods) {
                        Write-Host "Trying $($method.Name) with flags $($method.Flags)..." -ForegroundColor Yellow

                        # Clean up any existing file
                        if (Test-Path $tempPath) {
                            Remove-Item $tempPath -Force -ErrorAction SilentlyContinue
                        }

                        try {
                            $startTime = Get-Date
                            $shell.NameSpace($env:TEMP).CopyHere($opusFile, $method.Flags)

                            # Wait up to 10 seconds for copy
                            $timeout = 10
                            $copied = $false

                            for ($i = 0; $i -lt $timeout; $i++) {
                                Start-Sleep -Seconds 1
                                if (Test-Path $tempPath) {
                                    $tempFile = Get-Item $tempPath
                                    if ($tempFile.Length -gt 0) {
                                        $copied = $true
                                        break
                                    }
                                }
                            }

                            if ($copied) {
                                $endTime = Get-Date
                                $duration = ($endTime - $startTime).TotalSeconds
                                $tempFile = Get-Item $tempPath
                                $realSize = $tempFile.Length
                                $realSizeMB = [math]::Round($realSize/1MB, 2)

                                Write-Host "SUCCESS! Copy method worked!" -ForegroundColor Green
                                Write-Host "Real file size: $realSize bytes ($realSizeMB MB)" -ForegroundColor Green
                                Write-Host "Copy time: $duration seconds" -ForegroundColor Cyan
                                Write-Host "Transfer speed: $([math]::Round($realSize/1MB/$duration, 2)) MB/s" -ForegroundColor Cyan

                                # Keep this copy and exit
                                Write-Host "File saved to: $tempPath" -ForegroundColor White
                                break
                            } else {
                                Write-Host "Copy timeout or failed" -ForegroundColor Red
                            }
                        } catch {
                            Write-Host "Copy method failed: $($_.Exception.Message)" -ForegroundColor Red
                        }
                    }

                    # Clean up if we're just testing
                    if (Test-Path $tempPath) {
                        # Remove-Item $tempPath -Force -ErrorAction SilentlyContinue
                        Write-Host "Leaving test file at: $tempPath" -ForegroundColor Yellow
                    }
                } catch {
                    Write-Host "Stream reading failed: $($_.Exception.Message)" -ForegroundColor Red
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