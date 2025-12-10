# 获取USB设备信息的PowerShell脚本
# 用于查找录音笔的VID和PID

Write-Host "正在扫描USB设备..." -ForegroundColor Green
Write-Host "================================" -ForegroundColor Green

# 获取所有USB设备
$usbDevices = Get-WmiObject Win32_PnPEntity | Where-Object {
    $_.DeviceID -like "*USB*" -and
    ($_.Name -like "*录音*" -or
     $_.Name -like "*Record*" -or
     $_.Name -like "*Voice*" -or
     $_.Name -like "*SR*" -or
     $_.Name -like "*2207*" -or
     $_.Description -like "*便携*" -or
     $_.Description -like "*Storage*")
}

if ($usbDevices.Count -eq 0) {
    Write-Host "未找到录音笔设备，请确保设备已连接" -ForegroundColor Yellow
    Write-Host "显示所有便携式设备：" -ForegroundColor Cyan

    # 显示所有便携式设备
    $portableDevices = Get-WmiObject Win32_PnPEntity | Where-Object {
        $_.DeviceID -like "*USB*" -and
        ($_.Description -like "*便携*" -or
         $_.Description -like "*Storage*" -or
         $_.Name -like "*便携设备*")
    }

    foreach ($device in $portableDevices) {
        Write-Host "设备名称: $($device.Name)" -ForegroundColor White
        Write-Host "设备描述: $($device.Description)" -ForegroundColor Gray
        Write-Host "设备ID: $($device.DeviceID)" -ForegroundColor Yellow
        Write-Host "--------------------------------"
    }
}
else {
    foreach ($device in $usbDevices) {
        Write-Host "设备名称: $($device.Name)" -ForegroundColor White
        Write-Host "设备描述: $($device.Description)" -ForegroundColor Gray
        Write-Host "设备ID: $($device.DeviceID)" -ForegroundColor Yellow

        # 从DeviceID中提取VID和PID
        if ($device.DeviceID -match "VID_([0-9A-F]+).*PID_([0-9A-F]+)") {
            $vid = $matches[1]
            $pid = $matches[2]
            Write-Host "VID: $vid" -ForegroundColor Green
            Write-Host "PID: $pid" -ForegroundColor Green

            # 生成配置内容
            Write-Host "`n配置文件内容：" -ForegroundColor Cyan
            Write-Host "source:" -ForegroundColor White
            Write-Host "  device_name: `"$($device.Name)`"" -ForegroundColor Gray
            Write-Host "  vid: `"$($vid.ToLower())`"" -ForegroundColor Green
            Write-Host "  pid: `"$($pid.ToLower())`"" -ForegroundColor Green
        }
        Write-Host "--------------------------------"
    }
}

# 额外检查：查看便携式设备命名空间
Write-Host "`n检查便携式设备命名空间..." -ForegroundColor Green
try {
    $shell = New-Object -ComObject Shell.Application
    $portable = $shell.NameSpace(17)
    if ($portable) {
        $items = $portable.Items()
        Write-Host "便携式设备列表：" -ForegroundColor Cyan
        foreach ($item in $items) {
            Write-Host "  - $($item.Name)" -ForegroundColor White
        }
    }
}
catch {
    Write-Host "无法访问便携式设备命名空间" -ForegroundColor Red
}

Write-Host "`n按任意键继续..." -ForegroundColor Yellow
$null = $Host.UI.RawUI.ReadKey("NoEcho,IncludeKeyDown")