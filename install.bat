@echo off

set version=0.20.0

if "%PROCESSOR_ARCHITECTURE%"=="x86" (
    set binary_arch=386
) else (
    set binary_arch=amd64
)

set fzf_base=%cd%

rem Try to download binary executable
call :download fzf-%version%-windows_%binary_arch%.zip
echo.
echo For more information, see: https://github.com/junegunn/fzf
exit /b 0

rem check binary
:check_binary
echo     - Checking fzf executable ...
for /f "usebackq tokens=1" %%i in (`"%fzf_base%\bin\fzf.exe" --version`) do set output=%%i
if errorlevel 1 (
    echo Error: %output%
    set binary_error="Invalid binary"
) else (
    if %version% neq %output% (
        echo version: %output% != %version%
        set binary_error="Invalid version"
    ) else (
        echo version: %output%
        set binary_error=""
        exit /b 0
    )
)
del "%fzf_base%"\bin\fzf.exe
exit /b 1

rem try to use curl to download the file
:try_curl
set temp=%TMP%\fzf.zip
where curl >nul 2>&1
if errorlevel 1 (
    exit /b 1
) else (
    where 7z >nul 2>&1
    if errorlevel 1 (
        rem can't find 7-zip
        where winrar >nul 2>&1
        if errorlevel 1 (
            rem also can't find WinRAR
            echo Please intall 7-zip or WinRAR or put them into PATH if they has been installed!
        ) else (
            curl -fLo %temp% %~1 && winrar x %temp% && del /s /q %temp%
        )
    ) else (
        curl -fLo %temp% %~1 && 7z -y x %temp% && del /s /q %temp%
    )
    exit /b 0
)
goto :eof

rem try to use wget to download the file
:try_wget
set temp=%TMP%\fzf.zip
where wget >nul 2>&1
if errorlevel 1 (
    exit /b 1
) else (
    where 7z >nul 2>&1
    if errorlevel 1 (
        rem can't find 7-zip
        where winrar >nul 2>&1
        if errorlevel 1 (
            rem also can't find WinRAR
            echo Please intall 7-zip or WinRAR or put them into PATH if they has been installed!
        ) else (
            wget -O %temp% %~1 && winrar x %temp% && del /s /q %temp%
        )
    ) else (
        wget -O %temp% %~1 && 7z -y x %temp% && del /s /q %temp%
    )
    exit /b 0
)
goto :eof

rem download the file
:download
echo Downloading bin\fzf.exe ...
if not "%version%"=="alpha" (
    if exist "%fzf_base%\bin\fzf.exe" (
        echo     - Already exists
        call :check_binary && exit /b 0
    )
)
if not exist "%fzf_base%\bin" (
    mkdir "%fzf_base%\bin"
)
cd "%fzf_base%\bin"
if errorlevel 1 (
    set binary_error="Failed to create bin directory"
    exit /b 0
)
if "%version%"=="alpha" (
    set url=https://github.com/junegunn/fzf-bin/releases/download/alpha/%~1
) else (
    set url=https://github.com/junegunn/fzf-bin/releases/download/%version%/%~1
)
call :try_curl %url% || call :try_wget %url%
if errorlevel 1 (
    set binary_error="Failed to download with curl and wget"
    exit /b 0
)
if not exist fzf (
    set binary_error="Failed to download %~1"
    exit /b 0
)
call :check_binary
echo %binary_error%
goto :eof

