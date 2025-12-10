# 简化的设备测试脚本
Write-Host "=== 简化设备测试 ===" -ForegroundColor Green

try {
    $shell = New-Object -ComObject Shell.Application
    Write-Host "✅ Shell.Application 创建成功" -ForegroundColor Green

    $portable = $shell.NameSpace(17)
    if ($portable) {
        Write-Host "✅ 便携式设备命名空间获取成功" -ForegroundColor Green

        $devices = $portable.Items()
        Write-Host "找到 $($devices.Count) 个便携式设备" -ForegroundColor Cyan

        foreach ($device in $devices) {
            Write-Host "设备: $($device.Name)" -ForegroundColor White
            if ($device.Name -like "*SR302*") {
                Write-Host "  ✅ 找到SR302设备!" -ForegroundColor Green

                try {
                    $folder = $device.GetFolder()
                    Write-Host "  ✅ 设备文件夹访问成功" -ForegroundColor Green

                    $items = $folder.Items()
                    Write-Host "  设备包含 $($items.Count) 个项目" -ForegroundColor Cyan

                    foreach ($item in $items) {
                        Write-Host "    - $($item.Name) ($($item.Type))" -ForegroundColor Gray
                    }
                } catch {
                    Write-Host "  ❌ 无法访问设备文件夹: $($_.Exception.Message)" -ForegroundColor Red
                }
            }
        }
    } else {
        Write-Host "❌ 无法获取便携式设备命名空间" -ForegroundColor Red
    }
} catch {
    Write-Host "❌ 错误: $($_.Exception.Message)" -ForegroundColor Red
}

Write-Host "测试完成" -ForegroundColor Green