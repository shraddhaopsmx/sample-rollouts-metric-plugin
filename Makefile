# Build sample plugin with debug info
# https://www.jetbrains.com/help/go/attach-to-running-go-processes-with-debugger.html
.PHONY: build-sample-plugin-debug
build-sample-plugin-debug:
	go build -gcflags="all=-N -l" -o metric-plugin-linux-amd64 main.go

.PHONY: build-sample-plugin
build-sample-plugin:
	GOOS=linux GOARCH=amd64 go build -o metric-plugin-linux-amd64 main.go
	GOOS=linux GOARCH=arm64 go build -o metric-plugin-linux-arm64 main.go
	GOOS=darwin GOARCH=amd64 go build -o metric-plugin-darwin-amd64 main.go
	GOOS=darwin GOARCH=arm64 go build -o metric-plugin-darwin-arm64 main.go

