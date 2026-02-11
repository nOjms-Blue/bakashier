@ECHO OFF
SET BUILD_ERROR=0

where go >nul 2>&1
IF ERRORLEVEL 1 (
  ECHO Error: Go is not installed or not in PATH. Please install Go and try again.
	SET BUILD_ERROR=1
	@ECHO ON
  EXIT /B 1
)

go build -o bakashier.exe main.go
IF ERRORLEVEL 1 (
	SET BUILD_ERROR=1
	@ECHO ON
	EXIT /B %ERRORLEVEL%
)
@ECHO ON