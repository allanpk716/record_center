# Simple file size test script
Write-Host "=== File Size Test ===" -ForegroundColor Green

$shell = New-Object -ComObject Shell.Application
$portable = $shell.NameSpace(17)

if ($portable) {
    Write-Host "Got portable devices namespace" -ForegroundColor Green

    $device = $portable.Items() | Where-Object { $_.Name -like "*SR302*" } | Select-Object -First 1

    if ($device) {
        Write-Host "Found SR302 device: $($device.Name)" -ForegroundColor Green

        try {
            $deviceFolder = $device.GetFolder()
            Write-Host "Got device folder" -ForegroundColor Green

            function Find-OpusFiles($folder, $path = "") {
                $files = @()

                foreach ($item in $folder.Items()) {
                    if ($item.Name -like "*.opus") {
                        Write-Host "Found .opus file: $($item.Name)" -ForegroundColor Cyan

                        # Try to get size
                        $size = 0
                        $details = $folder.GetDetailsOf($item, 1)
                        Write-Host "  Details (index 1): '$details'" -ForegroundColor White

                        if ($details -match "(\d+)\s*(KB|MB|GB|B)") {
                            $num = [int]$matches[1]
                            $unit = $matches[2]
                            switch ($unit) {
                                "KB" { $size = $num * 1024 }
                                "MB" { $size = $num * 1024 * 1024 }
                                "GB" { $size = $num * 1024 * 1024 * 1024 }
                                "B"  { $size = $num }
                            }
                            Write-Host "  Parsed size: $size bytes" -ForegroundColor Green
                        } elseif ($details -match "^\d+$") {
                            $size = [int]$details
                            Write-Host "  Direct size: $size bytes" -ForegroundColor Green
                        }

                        # Try direct property
                        if ($size -eq 0 -and $item.Size) {
                            $size = [int]$item.Size
                            Write-Host "  Direct property: $size bytes" -ForegroundColor Green
                        }

                        # Default to 1MB if still 0
                        if ($size -eq 0) {
                            $size = 1048576  # 1MB
                            Write-Host "  Using default size: $size bytes" -ForegroundColor Yellow
                        }

                        Write-Host "  Final size: $($size/1MB) MB" -ForegroundColor Magenta

                        $fileInfo = @{
                            Name = $item.Name
                            Size = $size
                        }
                        $files += $fileInfo
                    } elseif ($item.IsFolder) {
                        try {
                            $subFolder = $item.GetFolder()
                            $files += Find-OpusFiles $subFolder "$path\$($item.Name)"
                        } catch {
                            Write-Host "Cannot access folder: $($item.Name)" -ForegroundColor Red
                        }
                    }
                }

                return $files
            }

            $opusFiles = Find-OpusFiles $deviceFolder
            Write-Host "Found $($opusFiles.Count) .opus files" -ForegroundColor Cyan

            foreach ($file in $opusFiles) {
                Write-Host "$($file.Name): $($file.Size/1MB) MB" -ForegroundColor White
            }

        } catch {
            Write-Host "Error accessing device folder: $($_.Exception.Message)" -ForegroundColor Red
        }
    } else {
        Write-Host "SR302 device not found" -ForegroundColor Red
    }
} else {
    Write-Host "Cannot get portable devices namespace" -ForegroundColor Red
}

Write-Host "Test completed" -ForegroundColor Green