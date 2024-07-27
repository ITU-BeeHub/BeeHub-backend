@echo off
:: Check if running as admin
net session >nul 2>&1
if %errorLevel% == 0 (
    echo Running with admin rights...
) else (
    echo Requesting admin rights...
    powershell -Command "Start-Process '%~f0' -Verb runAs"
    exit /b
)

:: Your commands here
sc.exe config BeeHubBotService start= auto
sc.exe start BeeHubBotService
