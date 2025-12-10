# Test SR302 device file access
Write-Host "=== SR302 File Test ===" -ForegroundColor Green

try {
    $shell = New-Object -ComObject Shell.Application
    $portable = $shell.NameSpace(17)

    if ($portable) {
        # Find SR302 device
        $device = $portable.Items() | Where-Object { $_.Name -eq "SR302" } | Select-Object -First 1

        if ($device) {
            Write-Host "Found SR302 device: $($device.Name)" -ForegroundColor Green
            Write-Host "Device path: $($device.Path)" -ForegroundColor Cyan

            try {
                $deviceFolder = $device.GetFolder()
                Write-Host "Successfully accessed device folder" -ForegroundColor Green

                # Function to recursively search for .opus files
                function Find-OpusFiles($folder, $path = "") {
                    $opusFiles = @()
                    $totalFiles = 0

                    Write-Host "Scanning folder: $($folder.Title)" -ForegroundColor Yellow

                    foreach ($item in $folder.Items()) {
                        $totalFiles++
                        $currentPath = if ($path -eq "") { $item.Name } else { "$path\$($item.Name)" }

                        if ($item.IsFolder) {
                            try {
                                $subFolder = $item.GetFolder()
                                if ($subFolder) {
                                    $subFiles = Find-OpusFiles $subFolder $currentPath
                                    $opusFiles += $subFiles.Files
                                    $totalFiles += $subFiles.Total
                                }
                            } catch {
                                Write-Host "Cannot access folder: $($item.Name)" -ForegroundColor Red
                            }
                        } elseif ($item.Name -like "*.opus") {
                            Write-Host "Found .opus file: $($item.Name)" -ForegroundColor Green

                            # Try to get file size with multiple methods
                            $size = 0
                            $sizeSource = "Unknown"

                            # Method 1: Direct Size property
                            if ($item.Size -and $item.Size -gt 0) {
                                $size = [long]$item.Size
                                $sizeSource = "Direct_Size"
                            }

                            # Method 2: Length property
                            if ($size -eq 0 -and $item.Length -and $item.Length -gt 0) {
                                $size = [long]$item.Length
                                $sizeSource = "Length_Property"
                            }

                            # Method 3: GetDetailsOf
                            if ($size -eq 0) {
                                try {
                                    $details = $folder.GetDetailsOf($item, 1)
                                    if ($details -and $details -match '(\d+(?:,\d+)*)\s*(KB|MB|GB|B)') {
                                        $numValue = $matches[1] -replace ',', ''
                                        $unit = $matches[2]
                                        switch ($unit) {
                                            "KB" { $size = [long][double]$numValue * 1024 }
                                            "MB" { $size = [long][double]$numValue * 1024 * 1024 }
                                            "GB" { $size = [long][double]$numValue * 1024 * 1024 * 1024 }
                                            "B"  { $size = [long][double]$numValue }
                                        }
                                        if ($size -gt 0) {
                                            $sizeSource = "Details_Property"
                                        }
                                    }
                                } catch {
                                    Write-Host "GetDetailsOf failed for $($item.Name)" -ForegroundColor Red
                                }
                            }

                            # Method 4: Intelligent estimation
                            if ($size -eq 0) {
                                $filename = $item.Name.ToLower()
                                if ($filename -match 'meeting|long') {
                                    $size = 100 * 1024 * 1024  # 100MB
                                    $sizeSource = "Meeting_Estimate"
                                } elseif ($filename -match 'memo|short') {
                                    $size = 3 * 1024 * 1024   # 3MB
                                    $sizeSource = "Memo_Estimate"
                                } else {
                                    $size = 7 * 1024 * 1024   # 7MB default
                                    $sizeSource = "Default_Estimate"
                                }
                            }

                            Write-Host "  Size: $([math]::Round($size/1MB, 2)) MB (Source: $sizeSource)" -ForegroundColor Cyan

                            $fileInfo = @{
                                Name = $item.Name
                                Path = $currentPath
                                Size = $size
                                SizeSource = $sizeSource
                                ModifiedDate = if ($item.ModifyDate) { $item.ModifyDate } else { [DateTime]::Now }
                            }
                            $opusFiles += $fileInfo
                        }
                    }

                    return @{
                        Files = $opusFiles
                        Total = $totalFiles
                    }
                }

                # Start searching
                $result = Find-OpusFiles $deviceFolder

                Write-Host "`n=== Results ===" -ForegroundColor Magenta
                Write-Host "Total items scanned: $($result.Total)" -ForegroundColor Cyan
                Write-Host ".opus files found: $($result.Files.Count)" -ForegroundColor Green

                if ($result.Files.Count -gt 0) {
                    Write-Host "`nFile details:" -ForegroundColor Yellow
                    foreach ($file in $result.Files) {
                        Write-Host "  $($file.Name)" -ForegroundColor White
                        Write-Host "    Path: $($file.Path)" -ForegroundColor Gray
                        Write-Host "    Size: $([math]::Round($file.Size/1MB, 2)) MB" -ForegroundColor Cyan
                        Write-Host "    Source: $($file.SizeSource)" -ForegroundColor Gray
                        Write-Host "    Modified: $($file.ModifiedDate)" -ForegroundColor Gray
                        Write-Host ""
                    }

                    $totalSize = ($result.Files | Measure-Object -Property Size -Sum).Sum
                    Write-Host "Total size: $([math]::Round($totalSize/1MB, 2)) MB" -ForegroundColor Green
                } else {
                    Write-Host "No .opus files found on SR302 device" -ForegroundColor Yellow
                    Write-Host "The device might be empty or files are stored in a different location" -ForegroundColor Yellow
                }

            } catch {
                Write-Host "Error accessing device folder: $($_.Exception.Message)" -ForegroundColor Red
            }
        } else {
            Write-Host "SR302 device not found in portable devices" -ForegroundColor Red
        }
    } else {
        Write-Host "Cannot access portable device namespace" -ForegroundColor Red
    }
} catch {
    Write-Host "Error: $($_.Exception.Message)" -ForegroundColor Red
}

Write-Host "Test completed" -ForegroundColor Green