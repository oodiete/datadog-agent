ARG BASE_IMAGE=mcr.microsoft.com/powershell:nanoserver-1809
FROM ${BASE_IMAGE}

LABEL maintainer "Datadog <package@datadoghq.com>"

COPY datadog-agent-7-latest.amd64.zip install.ps1 ./
RUN dir
RUN pwsh -Command "echo Hello Dockerfile"
RUN pwsh ./install.ps1

EXPOSE 8125/udp 8126/tcp

# Config file is choosen at runtime by the entrypoint script
COPY entrypoint.ps1 .
ADD entrypoint-ps1 ./entrypoint-ps1
COPY datadog*.yaml C:/ProgramData/Datadog/

ENTRYPOINT ["pwsh", "./entrypoint.ps1"]

CMD ["run"]
