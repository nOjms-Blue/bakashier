@ECHO OFF

REM Build bakashier.exe
cd %~dp0
cd ..
CALL scripts\build.bat
IF ERRORLEVEL 1 EXIT /B %ERRORLEVEL%
@ECHO OFF

REM Copy bakashier.exe
IF NOT EXIST "%LOCALAPPDATA%\bakashier" (
	MKDIR "%LOCALAPPDATA%\bakashier"
)
COPY /Y ".\bakashier.exe" "%LOCALAPPDATA%\bakashier\bakashier.exe" >nul 2>&1

REM Add to PATH
powershell -NoProfile -Command "$dir=[Environment]::ExpandEnvironmentVariables('%LOCALAPPDATA%\bakashier'); if(Test-Path $dir){ $p=[Environment]::GetEnvironmentVariable('Path','User'); $parts=$p -split ';'; if(-not ($parts | Where-Object { $_ -ieq $dir })) { [Environment]::SetEnvironmentVariable('Path', ($p + ';' + $dir).Trim(';'), 'User'); } }"

@ECHO ON
