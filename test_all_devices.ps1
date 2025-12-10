# 检查所有设备的PowerShell脚本
Write-Host "=== 检查所有设备 ===" -ForegroundColor Green

# 1. 检查便携式设备
Write-Host "`n1. 检查便携式设备 (Shell.Namespace(17)):" -ForegroundColor Cyan
try {
    $shell = New-Object -ComObject Shell.Application
    $portable = $shell.NameSpace(17)

    if ($portable) {
        $devices = $portable.Items()
        Write-Host "找到 $($devices.Count) 个便携式设备"

        foreach ($device in $devices) {
            Write-Host "  - 设备名: $($device.Name)" -ForegroundColor White
            Write-Host "    路径: $($device.Path)" -ForegroundColor Gray
        }
    } else {
        Write-Host "无法获取便携式设备命名空间"
    }
} catch {
    Write-Host "错误: $($_.Exception.Message)"
}

# 2. 检查桌面设备
Write-Host "`n2. 检查桌面设备:" -ForegroundColor Cyan
try {
    $shell = New-Object -ComObject Shell.Application
    $desktop = $shell.NameSpace(0)

    if ($desktop) {
        $desktopItems = $desktop.Items()
        foreach ($item in $desktopItems) {
            if ($item.Name -eq "This PC" -or $item.Name -eq "计算机") {
                Write-Host "找到计算机: $($item.Name)"
                $computerFolder = $item.GetFolder()

                foreach ($subItem in $computerFolder.Items()) {
                    if ($subItem.Name -like "*SR302*" -or $subItem.Type -like "*Portable*") {
                        Write-Host "  可能的目标设备: $($subItem.Name) (类型: $($subItem.Type))" -ForegroundColor Green
                    }
                }
            }
        }
    }
} catch {
    Write-Host "错误: $($_.Exception.Message)"
}

# 3. 使用WMI检查USB设备
Write-Host "`n3. WMI检查USB设备:" -ForegroundColor Cyan
try {
    $usbDevices = Get-WmiObject Win32_PnPEntity | Where-Object {
        $_.DeviceID -like "*USB*" -and
        $_.Name -like "*SR302*" -and
        $_.PNPClass -eq "WPD"
    }

    if ($usbDevices) {
        foreach ($device in $usbDevices) {
            Write-Host "  WPD设备: $($device.Name)" -ForegroundColor Green
            Write-Host "    设备ID: $($device.DeviceID)" -ForegroundColor Gray
            Write-Host "    PNP类: $($device.PNPClass)" -ForegroundColor Gray
            Write-Host "    状态: $($device.Status)" -ForegroundColor Gray
        }
    } else {
        Write-Host "未找到SR302 WPD设备"
    }
} catch {
    Write-Host "错误: $($_.Exception.Message)"
}

# 4. 使用PowerShell Windows Portable Device模块 (Windows 11)
Write-Host "`n4. Windows 11 MTP模块检查:" -ForegroundColor Cyan
try {
    # 检查是否支持Windows Portable Device模块
    $module = Get-Module -ListAvailable -Name "WindowsPortableDevice" -ErrorAction SilentlyContinue
    if ($module) {
        Write-Host "✅ 支持Windows Portable Device模块" -ForegroundColor Green
        $devices = Get-WindowsPortableDevice -ErrorAction SilentlyContinue
        if ($devices) {
            foreach ($device in $devices) {
                Write-Host "  设备: $($device.FriendlyName)" -ForegroundColor White
                Write-Host "    ID: $($device.DeviceId)" -ForegroundColor Gray
            }
        } else {
            Write-Host "未找到便携式设备"
        }
    } else {
        Write-Host "❌ 不支持Windows Portable Device模块 (可能不是Windows 11 24H2)"
    }
} catch {
    Write-Host "错误: $($_.Exception.Message)"
}

Write-Host "`n=== 设备检查完成 ===" -ForegroundColor Green
Write-Host "如果以上方法都无法找到SR302设备，请检查：" -ForegroundColor Yellow
Write-Host "1. 设备是否正确连接到USB端口" -ForegroundColor White
Write-Host "2. 设备是否在Windows文件管理器中可见" -ForegroundColor White
Write-Host "3. 设备驱动程序是否正确安装" -ForegroundColor White
Write-Host "4. 设备是否处于正确的模式（MTP/大容量存储）" -ForegroundColor White