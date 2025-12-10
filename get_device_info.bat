@echo off
chcp 65001 >nul
echo 正在获取USB设备信息...
echo ========================================

echo.
echo 方法1: 使用PowerShell查看录音笔相关设备
powershell -Command "Get-WmiObject Win32_PnPEntity | Where-Object {\$_.DeviceID -like '*USB*' -and (\$_.Name -like '*录音*' -or \$_.Name -like '*Record*' -or \$_.Name -like '*Voice*' -or \$_.Name -like '*SR302*' -or \$_.DeviceID -like '*VID_2207*')} | Format-List Name,Description,DeviceID"

echo.
echo ========================================
echo 方法2: 查看所有便携式设备
echo.
powershell -Command "Get-WmiObject Win32_PnPEntity | Where-Object {\$_.Description -like '*便携*' -or \$_.Description -like '*Storage*'} | Format-List Name,Description,DeviceID"

echo.
echo ========================================
echo 方法3: 查看所有USB存储设备
echo.
powershell -Command "Get-WmiObject Win32_PnPEntity | Where-Object {\$_.DeviceID -like '*USB*' -and (\$_.Description -like '*USB Mass Storage*' -or \$_.Description -like '*Storage Device*')} | Format-List Name,Description,DeviceID"

echo.
echo 如果找到了您的设备，VID和PID通常在DeviceID中显示为：VID_xxxx&PID_xxxx
echo 例如：USB\VID_2207&PID_0011...
echo.
pause