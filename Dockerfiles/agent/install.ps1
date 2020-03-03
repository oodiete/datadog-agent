msiexec /i datadog-agent-7-latest.amd64.msi /qn /L*v install.txt
start-process msiexec -ArgumentList '/i datadog-agent-7-latest.amd64.msi /qn /L*v install.txt' -Wait
cat install.txt
