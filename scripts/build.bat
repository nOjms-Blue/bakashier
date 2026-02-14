@ECHO OFF
where go >nul 2>&1
IF ERRORLEVEL 1 (
  ECHO Error: Go is not installed or not in PATH. Please install Go and try again.
  EXIT /B 1
)

go build -o bakashier.exe main.go
IF ERRORLEVEL 1 EXIT /B %ERRORLEVEL%
@ECHO ON
