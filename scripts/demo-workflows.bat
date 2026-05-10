@echo off
REM Atlas Flow - Demo Workflows Script with Authentication (Windows)
REM This script creates and executes demo workflows with proper user authentication

setlocal enabledelayedexpansion

REM Colors for output
set CYAN=[36m
set GREEN=[32m
set YELLOW=[33m
set BLUE=[34m
set MAGENTA=[35m
set RED=[31m
set NC=[0m

set API_URL=http://localhost:8002
set AUTH_URL=http://localhost:8001/auth
set DELAY=2

REM Test user credentials
set TEST_EMAIL=demo@atlasflow.local
set TEST_PASSWORD=demo-password-123

echo.
echo %CYAN%╔════════════════════════════════════════════════════════════════╗%NC%
echo %CYAN%║    Atlas Flow - Demo Workflows (With Authentication)          ║%NC%
echo %CYAN%║            Real Worker Orchestration ^& User Isolation         ║%NC%
echo %CYAN%╚════════════════════════════════════════════════════════════════╝%NC%
echo.

REM Check if API is ready
echo %BLUE%Checking if services are ready...%NC%
setlocal enabledelayedexpansion
set "count=0"
:check_api_loop
curl -s %API_URL%/health >nul 2>&1
if !errorlevel! equ 0 (
    echo %GREEN%✓ Orchestrator API is ready%NC%
    goto check_api_done
)
set /a count+=1
if !count! lss 30 (
    echo -n "."
    timeout /t 1 /nobreak >nul
    goto check_api_loop
)
echo %RED%✗ API is not responding%NC%
exit /b 1

:check_api_done

REM Authenticate and get token
echo.
echo %BLUE%Step 1: Authenticating user...%NC%

REM Try to login first
for /f "delims=" %%A in ('curl -s -X POST "%AUTH_URL%/login" -H "Content-Type: application/json" -d "{"email": "%TEST_EMAIL%", "password": "%TEST_PASSWORD%"}"') do set "LOGIN_RESPONSE=%%A"

echo %LOGIN_RESPONSE% | findstr /I "access_token" >nul
if !errorlevel! equ 0 (
    echo %GREEN%✓ Login successful%NC%
    for /f "tokens=3 delims=:," %%A in ('echo !LOGIN_RESPONSE! ^| findstr /o "access_token"') do (
        set "ACCESS_TOKEN=%%A"
    )
) else (
    echo %YELLOW%  User not found, creating new user...%NC%
    for /f "delims=" %%A in ('curl -s -X POST "%AUTH_URL%/signup" -H "Content-Type: application/json" -d "{"email": "%TEST_EMAIL%", "password": "%TEST_PASSWORD%", "full_name": "Demo User"}"') do set "SIGNUP_RESPONSE=%%A"
    
    echo !SIGNUP_RESPONSE! | findstr /I "access_token" >nul
    if !errorlevel! equ 0 (
        echo %GREEN%✓ User created and logged in%NC%
    ) else (
        echo %RED%✗ Failed to authenticate%NC%
        exit /b 1
    )
)

echo.
echo %GREEN%═══════════════════════════════════════════════════════════════%NC%
echo %GREEN%User Authentication ^& Isolation%NC%
echo %GREEN%═══════════════════════════════════════════════════════════════%NC%
echo.
echo %BLUE%Important:%NC%
echo   • All workflows are scoped to: %YELLOW%%TEST_EMAIL%%NC%
echo   • Workers are registered to: %YELLOW%demo-user%NC%
echo   • Only this user can see these workflows and workers
echo   • Another user's account will be isolated
echo.

echo %GREEN%═══════════════════════════════════════════════════════════════%NC%
echo %GREEN%DEMO 1: Simple Echo Pipeline (Sequential)%NC%
echo %GREEN%═══════════════════════════════════════════════════════════════%NC%
echo.

REM Demo 1 - Simple Echo Pipeline
set "DEMO1={\"name\": \"Echo Pipeline\", \"definition\": {\"tasks\": [{\"id\": \"greeting\", \"type\": \"echo\", \"payload\": {\"message\": \"Starting Atlas Flow demo...\"}}, {\"id\": \"status\", \"type\": \"echo\", \"payload\": {\"message\": \"All systems operational\"}, \"depends_on\": [\"greeting\"]}]}}"

echo %BLUE%Creating workflow...%NC%
for /f "delims=" %%A in ('curl -s -X POST "%API_URL%/workflows" -H "Content-Type: application/json" -H "Authorization: Bearer !ACCESS_TOKEN!" -d "!DEMO1!"') do set "RESPONSE=%%A"

echo %GREEN%✓ Workflow created%NC%

echo.
echo %BLUE%📊 Monitor System:%NC%
echo   • Orchestrator: %YELLOW%docker-compose logs -f workflow-service%NC%
echo   • All Workers:  %YELLOW%docker-compose logs -f worker-1 worker-2 worker-3%NC%
echo.

echo.
echo %CYAN%╔════════════════════════════════════════════════════════════════╗%NC%
echo %CYAN%║                      Demo Complete!                           ║%NC%
echo %CYAN%╚════════════════════════════════════════════════════════════════╝%NC%
echo.
echo %BLUE%🔐 Authentication ^& User Isolation:%NC%
echo   • Logged in as: %YELLOW%%TEST_EMAIL%%NC%
echo   • Workers belong to: %YELLOW%demo-user%NC%
echo   • All workflows are scoped to your account
echo   • Other users cannot see your workflows or workers
echo.
echo %BLUE%🔍 Check Workflows:%NC%
echo   curl -H "Authorization: Bearer !ACCESS_TOKEN!" %API_URL%/workflows ^| jq .
echo.
echo %BLUE%🔍 Check Workers (Isolated to User):%NC%
echo   curl -H "Authorization: Bearer !ACCESS_TOKEN!" %API_URL%/workers ^| jq .
echo.
