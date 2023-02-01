# Build sample plugin with debug info
# https://www.jetbrains.com/help/go/attach-to-running-go-processes-with-debugger.html
.PHONY: build-sample-plugin-debug
build-sample-plugin-debug:
	CGO_ENABLED=0 go build -gcflags="all=-N -l" -o metric-plugin main.go

.PHONY: build-sample-plugin
build-sample-plugin:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o metric-plugin-linux-amd64 main.go
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o metric-plugin-linux-arm64 main.go
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o metric-plugin-darwin-amd64 main.go
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o metric-plugin-darwin-arm64 main.go

