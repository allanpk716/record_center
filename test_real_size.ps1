# 测试真实的文件大小获取方法
Write-Host "=== 测试Windows文件管理器方法 ===" -ForegroundColor Green

# 测试1：直接访问已知的录音笔文件路径
Write-Host "`n测试1: 直接访问MTP设备文件路径" -ForegroundColor Cyan

# 通过文件管理器已知路径访问
$devicePath = "\\?\usb#vid_2207&pid_0011&mi_00#7&117ed41b&0&0000#{6ac27878-a6fa-4155-ba85-f98f491d4f33}\内部共享存储空间\录音笔文件\2025\11月\11月24日董总会谈录音_1\11月24日董总会谈录音_1.opus"

Write-Host "尝试访问: $devicePath" -ForegroundColor Yellow

try {
    $fileInfo = Get-Item $devicePath -ErrorAction SilentlyContinue
    if ($fileInfo) {
        Write-Host "✅ 直接访问成功!" -ForegroundColor Green
        Write-Host "  文件大小: $($fileInfo.Length) 字节 ($([math]::Round($fileInfo.Length/1MB, 2)) MB)" -ForegroundColor Green
        Write-Host "  修改时间: $($fileInfo.LastWriteTime)" -ForegroundColor White
        Write-Host "  文件属性: $($fileInfo.Attributes)" -ForegroundColor Gray
    } else {
        Write-Host "❌ 直接访问失败" -ForegroundColor Red
    }
} catch {
    Write-Host "❌ 直接访问异常: $($_.Exception.Message)" -ForegroundColor Red
}

# 测试2：通过文件系统获取盘符
Write-Host "`n测试2: 检查MTP设备是否分配了盘符" -ForegroundColor Cyan

# 尝试映射到盘符
$mtpDrive = Get-PSDrive -Name "MTP" -ErrorAction SilentlyContinue
if ($mtpDrive) {
    Write-Host "✅ 找到MTP盘符: $($mtpDrive.Root)" -ForegroundColor Green
    $testPath = "$($mtpDrive.Root)\内部共享存储空间\录音笔文件"
    Write-Host "检查路径: $testPath" -ForegroundColor Yellow
    if (Test-Path $testPath) {
        Get-ChildItem $testPath -Recurse -Filter *.opus | ForEach-Object {
            Write-Host "  文件: $($_.Name) - 大小: $($_.Length) 字节 ($([math]::Round($_.Length/1MB, 2)) MB)" -ForegroundColor White
        }
    }
} else {
    Write-Host "❌ 未找到MTP盘符" -ForegroundColor Red
}

# 测试3：检查所有可用的盘符
Write-Host "`n测试3: 扫描所有盘符寻找录音笔文件" -ForegroundColor Cyan

Get-PSDrive | Where-Object { $_.Used -or $_.Free } | ForEach-Object {
    Write-Host "检查盘符: $($_.Name) - 根目录: $($_.Root)" -ForegroundColor Yellow
    try {
        # 搜索.opus文件
        $opusFiles = Get-ChildItem $_.Root -Recurse -Filter *.opus -ErrorAction SilentlyContinue
        if ($opusFiles) {
            Write-Host "  ✅ 在 $($_.Name): 找到 $($opusFiles.Count) 个.opus文件" -ForegroundColor Green
            $opusFiles | Select-Object -First 5 | ForEach-Object {
                Write-Host "    - $($_.Name): $($_.Length) 字节 ($([math]::Round($_.Length/1MB, 2)) MB)" -ForegroundColor White
            }
        }
    } catch {
        Write-Host "  ❌ 扫描 $($_.Name): $($_.Exception.Message)" -ForegroundColor Red
    }
}

# 测试4：使用WMICache获取MTP设备信息
Write-Host "`n测试4: 通过WMI缓存获取MTP设备信息" -ForegroundColor Cyan

try {
    # 获取便携式设备
    $portableDevices = Get-CimInstance -ClassName Win32_PnPEntity | Where-Object { $_.PNPClass -eq "WPD" }
    if ($portableDevices) {
        Write-Host "✅ 找到便携式设备:" -ForegroundColor Green
        $portableDevices | ForEach-Object {
            Write-Host "  设备: $($device.Name)" -ForegroundColor White
            Write-Host "    设备ID: $($device.DeviceID)" -ForegroundColor Gray
            Write-Host "    制造商: $($device.Manufacturer)" -ForegroundColor Gray
        }
    } else {
        Write-Host "❌ 未找到便携式设备" -ForegroundColor Red
    }
} catch {
    Write-Host "❌ WMI查询失败: $($_.Exception.Message)" -ForegroundColor Red
}

# 测试5：尝试创建文件副本并检查大小
Write-Host "`n测试5: 创建临时副本验证文件大小" -ForegroundColor Cyan

if (Test-Path $devicePath) {
    $tempPath = "$env:TEMP\test_opus_file.opus"
    Write-Host "创建临时副本: $tempPath" -ForegroundColor Yellow
    try {
        Copy-Item $devicePath $tempPath -ErrorAction Stop
        $tempInfo = Get-Item $tempPath
        Write-Host "✅ 副本创建成功!" -ForegroundColor Green
        Write-Host "  原文件大小: $(Get-Item $devicePath).Length 字节" -ForegroundColor Cyan
        Write-Host "  副本大小: $($tempInfo.Length) 字节" -ForegroundColor Green

        # 清理
        Remove-Item $tempPath -ErrorAction SilentlyContinue
        Write-Host "  已清理临时文件" -ForegroundColor Gray
    } catch {
        Write-Host "❌ 副本失败: $($_.Exception.Message)" -ForegroundColor Red
    }
} else {
    Write-Host "❌ 源文件不存在，无法测试" -ForegroundColor Red
}

Write-Host "`n=== 测试完成 ===" -ForegroundColor Green