@rem param1 GOPATH
@rem param2 build path

@SET GOPATH=%1
@CD %2
@go build

@rem succeed or failed
@if %errorlevel%==0 (echo build success) else (echo build failed)