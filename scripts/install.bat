@ECHO OFF

REM Build bakashier.exe
cd %~dp0
cd ..
SET NO_NEED_ECHO_ON=TRUE
CALL scripts\build.bat
@ECHO OFF
IF %BUILD_ERROR% EQU 1 (
	@ECHO ON
	EXIT /B 1
)

REM Copy bakashier.exe
IF NOT EXIST "%LOCALAPPDATA%\bakashier" (
	MKDIR "%LOCALAPPDATA%\bakashier"
)
COPY /Y ".\bakashier.exe" "%LOCALAPPDATA%\bakashier\bakashier.exe" >nul 2>&1
COPY /Y ".\LICENSE" "%LOCALAPPDATA%\bakashier\LICENSE" >nul 2>&1
COPY /Y ".\NOTICE" "%LOCALAPPDATA%\bakashier\NOTICE" >nul 2>&1
COPY /Y ".\README.md" "%LOCALAPPDATA%\bakashier\README.md" >nul 2>&1
COPY /Y ".\README.ja.md" "%LOCALAPPDATA%\bakashier\README.ja.md" >nul 2>&1

REM Add to PATH
powershell -NoProfile -Command "$dir=[Environment]::ExpandEnvironmentVariables('%LOCALAPPDATA%\bakashier'); if(Test-Path $dir){ $p=[Environment]::GetEnvironmentVariable('Path','User'); $parts=$p -split ';'; if(-not ($parts | Where-Object { $_ -ieq $dir })) { [Environment]::SetEnvironmentVariable('Path', ($p + ';' + $dir).Trim(';'), 'User'); } }"

@ECHO ON
