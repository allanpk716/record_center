# 测试设备文件枚举的PowerShell脚本
Write-Host "=== 测试SR302设备文件枚举 ===" -ForegroundColor Green

# 获取便携式设备命名空间
$shell = New-Object -ComObject Shell.Application
$portable = $shell.NameSpace(17)

if ($portable) {
    Write-Host "✅ 成功获取便携式设备命名空间" -ForegroundColor Green

    # 查找SR302设备
    $device = $portable.Items() | Where-Object { $_.Name -like "*SR302*" } | Select-Object -First 1

    if ($device) {
        Write-Host "✅ 找到SR302设备: $($device.Name)" -ForegroundColor Green
        Write-Host "设备路径: $($device.Path)" -ForegroundColor Cyan

        try {
            $deviceFolder = $device.GetFolder()
            Write-Host "✅ 成功获取设备文件夹" -ForegroundColor Green

            # 递归查找所有文件
            function Find-AllFiles($folder, $depth = 0) {
                $indent = "  " * $depth
                $totalFiles = 0
                $opusFiles = 0

                Write-Host "$indent扫描文件夹: $($folder.Title)" -ForegroundColor Yellow

                foreach ($item in $folder.Items()) {
                    if ($item.IsFolder) {
                        try {
                            $subFolder = $item.GetFolder()
                            if ($subFolder) {
                                $subStats = Find-AllFiles $subFolder ($depth + 1)
                                $totalFiles += $subStats.TotalFiles
                                $opusFiles += $subStats.OpusFiles
                            }
                        } catch {
                            Write-Host "$indent  无法访问文件夹: $($item.Name)" -ForegroundColor Red
                        }
                    } else {
                        $totalFiles++
                        Write-Host "$indent  文件: $($item.Name)" -ForegroundColor White

                        if ($item.Name -like "*.opus") {
                            $opusFiles++
                            Write-Host "$indent    ✅ .opus文件!" -ForegroundColor Green

                            # 尝试获取大小
                            $size = 0
                            try {
                                if ($item.Size -and $item.Size -gt 0) {
                                    $size = [long]$item.Size
                                    Write-Host "$indent    大小: $($size/1MB) MB (实际)" -ForegroundColor Green
                                } else {
                                    # 使用智能估算
                                    $filename = $item.Name.ToLower()
                                    if ($filename -match 'meeting|会议') {
                                        $size = 100 * 1024 * 1024  # 100MB
                                    } elseif ($filename -match 'memo|备忘') {
                                        $size = 3 * 1024 * 1024   # 3MB
                                    } else {
                                        $size = 7 * 1024 * 1024   # 7MB
                                    }
                                    Write-Host "$indent    大小: $($size/1MB) MB (智能估算)" -ForegroundColor Yellow
                                }
                            } catch {
                                Write-Host "$indent    无法获取文件大小" -ForegroundColor Red
                            }
                        }
                    }
                }

                return @{
                    TotalFiles = $totalFiles
                    OpusFiles = $opusFiles
                }
            }

            # 开始扫描
            $stats = Find-AllFiles $deviceFolder

            Write-Host "`n=== 扫描结果 ===" -ForegroundColor Magenta
            Write-Host "总文件数: $($stats.TotalFiles)" -ForegroundColor Cyan
            Write-Host ".opus文件数: $($stats.OpusFiles)" -ForegroundColor Green

            if ($stats.OpusFiles -eq 0) {
                Write-Host "`n⚠️  未找到.opus文件，可能的原因：" -ForegroundColor Yellow
                Write-Host "  1. 设备中没有录音文件" -ForegroundColor White
                Write-Host "  2. 录音文件存储在其他位置" -ForegroundColor White
                Write-Host "  3. 文件格式不是.opus" -ForegroundColor White
                Write-Host "  4. 设备访问权限问题" -ForegroundColor White
            }

        } catch {
            Write-Host "❌ 访问设备文件夹失败: $($_.Exception.Message)" -ForegroundColor Red
        }
    } else {
        Write-Host "❌ 未找到SR302设备" -ForegroundColor Red

        # 列出所有便携式设备
        Write-Host "`n所有便携式设备:" -ForegroundColor Cyan
        $portable.Items() | ForEach-Object {
            Write-Host "  - $($_.Name)" -ForegroundColor White
        }
    }
} else {
    Write-Host "❌ 无法获取便携式设备命名空间" -ForegroundColor Red
}

Write-Host "`n=== 测试完成 ===" -ForegroundColor Green