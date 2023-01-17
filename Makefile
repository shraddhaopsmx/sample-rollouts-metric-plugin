# Build sample plugin with debug info
# https://www.jetbrains.com/help/go/attach-to-running-go-processes-with-debugger.html
.PHONY: build-sample-plugin-debug
build-sample-plugin-debug:
	go build -gcflags="all=-N -l" -o metric-plugin main.go

.PHONY: build-sample-plugin
build-sample-plugin:
	go build -o metric-plugin main.go

