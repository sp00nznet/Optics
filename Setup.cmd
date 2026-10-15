@echo off
rem Quick start: checks for a C compiler and CMake, offers to install them,
rem builds Optics and leaves an Optics shortcut in this folder. Rerun it any time.
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0tools\setup.ps1"
if errorlevel 1 (echo. & echo Setup did not finish. See setup.log in this folder.)
pause
