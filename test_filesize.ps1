# 测试录音笔文件大小获取的PowerShell脚本

Write-Host "=== 录音笔文件大小测试 ===" -ForegroundColor Green

# 获取便携式设备命名空间
$shell = New-Object -ComObject Shell.Application
$portable = $shell.NameSpace(17)

if ($portable) {
    Write-Host "✅ 成功获取便携式设备命名空间" -ForegroundColor Green

    # 查找SR302设备
    $device = $portable.Items() | Where-Object { $_.Name -like "*SR302*" } | Select-Object -First 1

    if ($device) {
        Write-Host "✅ 找到SR302设备: $($device.Name)" -ForegroundColor Green

        try {
            $deviceFolder = $device.GetFolder()
            Write-Host "✅ 成功获取设备文件夹" -ForegroundColor Green

            # 递归查找.opus文件
            function Find-OpusFiles($folder, $path = "") {
                $opusFiles = @()

                Write-Host "正在扫描文件夹: $($folder.Title)" -ForegroundColor Yellow

                foreach ($item in $folder.Items()) {
                    $currentPath = if ($path -eq "") { $item.Name } else { "$path\$($item.Name)" }

                    if ($item.IsFolder) {
                        try {
                            $subFolder = $item.GetFolder()
                            if ($subFolder) {
                                $opusFiles += Find-OpusFiles $subFolder $currentPath
                            }
                        } catch {
                            Write-Host "无法访问文件夹: $($item.Name)" -ForegroundColor Red
                        }
                    } elseif ($item.Name -like "*.opus") {
                        Write-Host "`n发现.opus文件: $($item.Name)" -ForegroundColor Cyan

                        # 方法1: GetDetailsOf 获取大小
                        $size1 = 0
                        $details = $folder.GetDetailsOf($item, 1)
                        Write-Host "  方法1 (GetDetailsOf索引1): '$details'" -ForegroundColor White

                        if ($details) {
                            if ($details -match "([\d,]+)\s*(KB|MB|GB|B)") {
                                $numValue = $matches[1] -replace ",", ""
                                $unit = $matches[2]
                                switch ($unit) {
                                    "KB" { $size1 = [long][double]$numValue * 1024 }
                                    "MB" { $size1 = [long][double]$numValue * 1024 * 1024 }
                                    "GB" { $size1 = [long][double]$numValue * 1024 * 1024 * 1024 }
                                    "B"  { $size1 = [long][double]$numValue }
                                }
                                Write-Host "  解析结果: $size1 字节" -ForegroundColor Green
                            } elseif ($details -match "^\d+$") {
                                $size1 = [long]$details
                                Write-Host "  直接数字: $size1 字节" -ForegroundColor Green
                            }
                        }

                        # 方法2: 尝试其他索引
                        $size2 = 0
                        $details2 = $folder.GetDetailsOf($item, 0)  # 名称
                        $details3 = $folder.GetDetailsOf($item, 2)  # 类型
                        $details4 = $folder.GetDetailsOf($item, 3)  # 修改日期

                        Write-Host "  方法2 (其他索引):" -ForegroundColor White
                        Write-Host "    索引0 (名称): '$details2'" -ForegroundColor Gray
                        Write-Host "    索引2 (类型): '$details3'" -ForegroundColor Gray
                        Write-Host "    索引3 (修改日期): '$details4'" -ForegroundColor Gray

                        # 方法3: 直接属性
                        $size3 = 0
                        try {
                            if ($item.Size) {
                                $size3 = [long]$item.Size
                                Write-Host "  方法3 (直接Size属性): $size3 字节" -ForegroundColor Green
                            } else {
                                Write-Host "  方法3 (直接Size属性): 无法获取" -ForegroundColor Red
                            }
                        } catch {
                            Write-Host "  方法3 (直接Size属性): 出错 - $($_.Exception.Message)" -ForegroundColor Red
                        }

                        # 方法4: Length属性
                        $size4 = 0
                        try {
                            if ($item.Length) {
                                $size4 = [long]$item.Length
                                Write-Host "  方法4 (Length属性): $size4 字节" -ForegroundColor Green
                            } else {
                                Write-Host "  方法4 (Length属性): 无法获取" -ForegroundColor Red
                            }
                        } catch {
                            Write-Host "  方法4 (Length属性): 出错 - $($_.Exception.Message)" -ForegroundColor Red
                        }

                        # 选择最佳大小
                        $finalSize = $size1
                        if ($finalSize -eq 0 -and $size2 -gt 0) { $finalSize = $size2 }
                        if ($finalSize -eq 0 -and $size3 -gt 0) { $finalSize = $size3 }
                        if ($finalSize -eq 0 -and $size4 -gt 0) { $finalSize = $size4 }
                        if ($finalSize -eq 0) { $finalSize = 1024 * 1024 }  # 默认1MB

                        Write-Host "  最终文件大小: $finalSize 字节 ($([math]::Round($finalSize/1MB, 2)) MB)" -ForegroundColor Magenta

                        $fileInfo = [PSCustomObject]@{
                            Name = $item.Name
                            Path = $currentPath
                            Size = $finalSize
                            ModifiedDate = if ($item.ModifyDate) { $item.ModifyDate } else { [DateTime]::Now }
                            Methods = @{
                                Method1 = $size1
                                Method3 = $size3
                                Method4 = $size4
                                Details = $details
                            }
                        }

                        $opusFiles += $fileInfo
                    }
                }

                return $opusFiles
            }

            $opusFiles = Find-OpusFiles $deviceFolder

            Write-Host "`n=== 文件大小统计 ===" -ForegroundColor Green
            Write-Host "找到 $($opusFiles.Count) 个.opus文件" -ForegroundColor Cyan

            $totalSize = 0
            foreach ($file in $opusFiles) {
                $totalSize += $file.Size
                Write-Host "文件: $($file.Name)" -ForegroundColor White
                Write-Host "  路径: $($file.Path)" -ForegroundColor Gray
                Write-Host "  大小: $($file.Size) 字节 ($([math]::Round($file.Size/1MB, 2)) MB)" -ForegroundColor Yellow
                Write-Host "  修改时间: $($file.ModifiedDate)" -ForegroundColor Gray
                Write-Host ""
            }

            Write-Host "总大小: $totalSize 字节 ($([math]::Round($totalSize/1MB, 2)) MB)" -ForegroundColor Magenta

        } catch {
            Write-Host "❌ 访问设备文件夹失败: $($_.Exception.Message)" -ForegroundColor Red
        }
    } else {
        Write-Host "❌ 未找到SR302设备" -ForegroundColor Red
    }
} else {
    Write-Host "❌ 无法获取便携式设备命名空间" -ForegroundColor Red
}

Write-Host "`n=== 测试完成 ===" -ForegroundColor Green