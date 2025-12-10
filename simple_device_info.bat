@echo off
chcp 65001 >nul
echo 获取录音笔设备信息
echo ===================

echo.
echo 1. 使用PowerShell查看所有USB设备
echo.
powershell -Command "Get-WmiObject Win32_PnPEntity | Where-Object {$_.DeviceID -like '*USB*' -and $_.DeviceID -like '*VID_*PID_*'} | Select-Object Name, Description, DeviceID | Format-Table -AutoSize"

echo.
echo ===================
echo 2. 查看包含录音/录音笔的设备
echo.
powershell -Command "Get-WmiObject Win32_PnPEntity | Where-Object {$_.DeviceID -like '*USB*' -and ($_.Name -like '*录音*' -or $_.Description -like '*录音*' -or $_.Name -like '*SR302*')} | Select-Object Name, Description, DeviceID | Format-Table -AutoSize"

echo.
echo ===================
echo 3. 显示当前备份程序的检测结果
echo.
echo 您的程序已经检测到：
echo 设备名称: SR302
echo VID: 2207
echo PID: 0011
echo.
echo 这应该就是您的录音笔信息！

pause