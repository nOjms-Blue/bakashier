@ECHO OFF
SETLOCAL

REM Move to repository root
cd %~dp0
cd ..

REM Build bakashier.exe
SET NO_NEED_ECHO_ON=TRUE
CALL scripts\build.bat
@ECHO OFF
IF %BUILD_ERROR% EQU 1 (
	ENDLOCAL
	@ECHO ON
	EXIT /B 1
)

REM Prepare distribution directory
SET "DIST_DIR=dist"
IF EXIST "%DIST_DIR%" (
	RMDIR /S /Q "%DIST_DIR%"
)
MKDIR "%DIST_DIR%"
IF ERRORLEVEL 1 (
	ENDLOCAL
	@ECHO ON
	EXIT /B 1
)

REM Copy distributable files
COPY /Y ".\bakashier.exe" "%DIST_DIR%\bakashier.exe" >nul
COPY /Y ".\LICENSE" "%DIST_DIR%\LICENSE" >nul
COPY /Y ".\THIRD_PARTY_LICENSES.md" "%DIST_DIR%\THIRD_PARTY_LICENSES.md" >nul
COPY /Y ".\README.md" "%DIST_DIR%\README.md" >nul
COPY /Y ".\README.ja.md" "%DIST_DIR%\README.ja.md" >nul
XCOPY ".\third_party_licenses" "%DIST_DIR%\third_party_licenses" /E /I /H /Y >nul
IF ERRORLEVEL 1 (
	ENDLOCAL
	@ECHO ON
	EXIT /B 1
)

ECHO Created "%DIST_DIR%"
ENDLOCAL
@ECHO ON
