@echo off

rem BUILD REQUIREMENTS
rem Both GopherJS and Terser (via Node.js with npx) must both be installed and in your path
rem Make sure you set your GOPHERJS_GOROOT environmental variable as needed
rem set GOPHERJS_GOROOT=C:\path-to-gopherjs-goroot

set app=TigerShare

cd src

:ecmascript
echo Building JS...
set GOOS=js
set GOARCH=ecmascript
call gopherjs build -o ../Release/%app%.js
echo Minifying JS...
call npx terser ../Release/%app%.js -c -m -o ../Release/%app%.min.js

:html
if not exist ..\Release\index.html (
	copy index.html ..\Release
)

:server
set GOOS=
set GOARCH=
if not exist ..\Release\server.exe (
	echo Building server.exe...
	call go build -ldflags="-s -w" -o ../Release/server.exe
)

:done
del ..\Release\%app%.js ..\Release\%app%.js.map
echo Done
pause
exit /b
