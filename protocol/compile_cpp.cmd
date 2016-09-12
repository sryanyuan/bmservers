@pushd %~dp0

@set OUTPUT=.\
@if not exist %OUTPUT%		(mkdir %OUTPUT%)

@set _ITEM_= 
@for %%F in (*.proto) do @(
	@set _ITEM_=%%F
	@echo .		%%F
	@..\tools\protoc.exe %%F --cpp_out=%OUTPUT%
	@cd %~dp0
)

@pause

@popd