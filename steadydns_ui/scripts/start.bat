@echo off
REM Start development server script for Windows

REM Change to project root
cd /d "%~dp0.."

echo Starting SteadyDNS UI development server...

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

REM Start development server
echo Starting development server...
call npm run dev

exit /b %errorlevel%