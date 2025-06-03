@echo off
echo =================================
echo RemoteDesk Cliente - Admin Mode
echo =================================
echo.
echo Este archivo ejecuta el cliente con permisos de administrador
echo para asegurar que los clicks del mouse funcionen correctamente.
echo.
echo Ubicacion del ejecutable:
echo %~dp0build\bin\EscritorioRemoto-Cliente.exe
echo.
echo Presiona cualquier tecla para continuar...
pause >nul

echo.
echo Ejecutando cliente como administrador...
echo.

PowerShell -Command "Start-Process '%~dp0build\bin\EscritorioRemoto-Cliente.exe' -Verb RunAs"

echo.
echo Cliente iniciado. Verifica que se abrió la ventana del cliente.
echo Si no funciona, ejecuta manualmente como administrador:
echo 1. Click derecho en EscritorioRemoto-Cliente.exe
echo 2. Seleccionar "Ejecutar como administrador"
echo.
pause 