$ErrorActionPreference = "Stop"

Set-Location $PSScriptRoot

go mod download
go build -o bin/web-security-proxy.exe ./cmd/server
Write-Host "Build complete: bin/web-security-proxy.exe"
