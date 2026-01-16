@echo off
REM Build project script for Windows

REM Change to project root
cd /d "%~dp0.."

echo Building SteadyDNS UI project...

REM Check if npm is available
where npm >nul 2>&1
if %errorlevel% neq 0 (
    echo Error: npm is not installed. Please install Node.js first.
    pause
    exit /b 1
)

REM Install dependencies if needed
if not exist "node_modules" (
    echo Installing dependencies...
    call npm install
    if %errorlevel% neq 0 (
        echo Error: Failed to install dependencies.
        pause
        exit /b 1
    )
)

REM Build project
echo Building project...
call npm run build

if %errorlevel% equ 0 (
    echo Build completed successfully!
    echo Output directory: dist/
) else (
    echo Error: Build failed.
    pause
    exit /b 1
)

pause
exit /b 0