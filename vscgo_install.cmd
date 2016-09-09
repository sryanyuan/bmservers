@rem param1 GOPATH
@rem param2 build path

@SET GOPATH=%1
@CD %2
@go install

@rem succeed or failed
@if %errorlevel%==0 (echo install success) else (echo install failed)