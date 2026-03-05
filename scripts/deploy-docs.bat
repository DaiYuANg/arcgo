@echo off
REM 构建和部署文档脚本 (Windows)

echo arcgo 文档部署脚本
echo.

:menu
echo 请选择操作:
echo 1. 构建文档
echo 2. 本地预览
echo 3. 部署到 GitHub Pages
echo 4. 退出
echo.
set /p choice="请输入选项 (1-4): "

if "%choice%"=="1" goto build
if "%choice%"=="2" goto serve
if "%choice%"=="3" goto deploy
if "%choice%"=="4" goto end
echo 无效选项，请重试
goto menu

:build
echo 正在构建文档...
call mkdocs build --clean
echo 文档构建完成！输出目录：site/
pause
goto menu

:serve
echo 启动本地预览服务器...
echo 访问：http://127.0.0.1:8000
echo 按 Ctrl+C 停止
call mkdocs serve
goto menu

:deploy
echo 部署到 GitHub Pages...
pip install mike 2>nul
call mike deploy --push --update-aliases main latest
echo 部署完成！
pause
goto menu

:end
echo 再见！
exit /b
