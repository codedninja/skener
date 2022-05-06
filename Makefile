agent:
	GOOS=windows GOARCH=amd64 go build -ldflags "-H windowsgui" -o build/agent.exe cmd/agent/main.go

worker:
	CGO_ENABLED=0 go build -o build/worker cmd/worker/main.go

standalone:
	CGO_ENABLED=0 go build -o build/standalone-cli cmd/standalone-cli/main.go
