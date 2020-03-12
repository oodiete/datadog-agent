
Get-ChildItem 'entrypoint-ps1' | ForEach-Object {
	& $_.FullName
	if (-Not $?) {
		exit 1
	}
}

# Add java to the path for jmxfetch
setx PATH "$Env:Path;C:/java"
$Env:Path="$Env:Path;C:/java"

return & "C:/Program Files/Datadog/Datadog Agent/bin/agent.exe" $args
