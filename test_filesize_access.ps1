# 测试MTP设备文件大小获取方法
Write-Host "=== 测试Windows文件管理器获取MTP文件大小的方法 ===" -ForegroundColor Green

# 方法1：直接访问已知MTP设备路径
Write-Host "`n方法1: 直接访问MTP设备路径" -ForegroundColor Cyan

$knownPath = "\\?\usb#vid_2207&pid_0011&mi_00#7&117ed41b&0&0000#{6ac27878-a6fa-4155-ba85-f98f491d4f33}\内部共享存储空间\录音笔文件"
Write-Host "尝试路径: $knownPath" -ForegroundColor Yellow

if (Test-Path $knownPath) {
    try {
        $files = Get-ChildItem $knownPath -Recurse -Filter *.opus -ErrorAction Stop
        Write-Host "找到 $($files.Count) 个.opus文件" -ForegroundColor Green

        foreach ($file in $files) {
            Write-Host "文件: $($file.Name)" -ForegroundColor White
            Write-Host "  大小: $($file.Length) 字节 ($([math]::Round($file.Length/1MB, 2)) MB)" -ForegroundColor Cyan
            Write-Host "  修改时间: $($file.LastWriteTime)" -ForegroundColor Gray
        }
    } catch {
        Write-Host "访问失败: $($_.Exception.Message)" -ForegroundColor Red
    }
} else {
    Write-Host "路径不存在" -ForegroundColor Red
}

# 方法2：通过Shell.Application获取详细信息
Write-Host "`n方法2: Shell.Application详细信息获取" -ForegroundColor Cyan

try {
    $shell = New-Object -ComObject Shell.Application
    $portable = $shell.NameSpace(17)

    if ($portable) {
        $device = $portable.Items() | Where-Object { $_.Name -eq "SR302" } | Select-Object -First 1

        if ($device) {
            Write-Host "找到SR302设备" -ForegroundColor Green

            $deviceFolder = $device.GetFolder()
            $recordFolder = $deviceFolder.ParseName("内部共享存储空间\录音笔文件")

            if ($recordFolder) {
                $recordFolderObj = $recordFolder.GetFolder()
                Write-Host "成功访问录音文件夹" -ForegroundColor Green

                foreach ($item in $recordFolderObj.Items()) {
                    if ($item.Name -like "*.opus") {
                        Write-Host "文件: $($item.Name)" -ForegroundColor White

                        # 尝试多种方法获取文件大小
                        $size = 0
                        $method = ""

                        # 方法2a: 直接Size属性
                        if ($item.Size -gt 0) {
                            $size = $item.Size
                            $method = "Size属性"
                        }
                        # 方法2b: GetDetailsOf方法
                        else {
                            $details = $recordFolderObj.GetDetailsOf($item, 1)
                            if ($details -match '(\d+(?:,\d+)*)\s*(KB|MB|GB|B)') {
                                $num = $matches[1] -replace ',', ''
                                $unit = $matches[2]
                                $size = switch ($unit) {
                                    "KB" { [long][double]$num * 1024 }
                                    "MB" { [long][double]$num * 1024 * 1024 }
                                    "GB" { [long][double]$num * 1024 * 1024 * 1024 }
                                    "B"  { [long][double]$num }
                                }
                                $method = "GetDetailsOf"
                            }
                        }

                        Write-Host "  大小: $size 字节 ($([math]::Round($size/1MB, 2)) MB) - 方法: $method" -ForegroundColor Cyan
                    }
                }
            } else {
                Write-Host "无法找到录音文件夹" -ForegroundColor Red
            }
        } else {
            Write-Host "未找到SR302设备" -ForegroundColor Red
        }
    } else {
        Write-Host "无法访问便携式设备命名空间" -ForegroundColor Red
    }
} catch {
    Write-Host "Shell COM访问失败: $($_.Exception.Message)" -ForegroundColor Red
}

# 方法3：使用Windows 11的Get-WindowsPortableDevice（如果支持）
Write-Host "`n方法3: Windows 11原生MTP支持" -ForegroundColor Cyan

try {
    $modules = Get-Module -ListAvailable | Where-Object { $_.Name -like "*Portable*" }
    if ($modules) {
        Write-Host "发现便携式设备模块: $($modules.Name)" -ForegroundColor Green

        # 尝试导入并使用
        Import-Module WindowsPortableDevice -ErrorAction SilentlyContinue
        if (Get-Command Get-WindowsPortableDevice -ErrorAction SilentlyContinue) {
            Write-Host "使用Windows 11原生MTP API" -ForegroundColor Green
            $devices = Get-WindowsPortableDevice
            Write-Host "找到设备: $devices" -ForegroundColor White
        } else {
            Write-Host "无法导入WindowsPortableDevice模块" -ForegroundColor Yellow
        }
    } else {
        Write-Host "系统不支持Windows 11原生MTP模块" -ForegroundColor Yellow
    }
} catch {
    Write-Host "Windows 11 MTP测试失败: $($_.Exception.Message)" -ForegroundColor Red
}

# 方法4：创建文件副本测试真实大小
Write-Host "`n方法4: 文件副本大小验证" -ForegroundColor Cyan

if (Test-Path $knownPath) {
    try {
        $files = Get-ChildItem $knownPath -Recurse -Filter *.opus | Select-Object -First 1
        if ($files) {
            $file = $files[0]
            $tempPath = "$env:TEMP\test_copy.opus"

            Write-Host "复制文件: $($file.Name)" -ForegroundColor Yellow
            Write-Host "源文件大小: $($file.Length) 字节" -ForegroundColor Cyan

            Copy-Item $file.FullName $tempPath -ErrorAction Stop
            $copiedFile = Get-Item $tempPath

            Write-Host "副本文件大小: $($copiedFile.Length) 字节" -ForegroundColor Green
            Write-Host "大小是否一致: $(if ($file.Length -eq $copiedFile.Length) { '是' } else { '否' })" -ForegroundColor White

            # 清理
            Remove-Item $tempPath -ErrorAction SilentlyContinue
        } else {
            Write-Host "没有找到可用于测试的.opus文件" -ForegroundColor Yellow
        }
    } catch {
        Write-Host "文件副本测试失败: $($_.Exception.Message)" -ForegroundColor Red
    }
}

Write-Host "`n=== 测试完成 ===" -ForegroundColor Green