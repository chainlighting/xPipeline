.PHONY: all clean freebsd linux mac pi win current vendor test
#BUILD_FLAGS=GO15VENDOREXPERIMENT=1 GORACE="halt_on_error=0" GOGC=off
clean:
	@rm -f ./xPipeline
	@rm -f ./dist/xPipeline_*.tar.gz
	@go clean
    
linux:
	@echo "Building for Linux"
	@GOOS=linux GOARCH=amd64 $(BUILD_FLAGS) go build -o xPipeline
	@tar -zcf dist/xPipeline_linux.tar.gz xPipeline pipeline/*.yml

mac:
	@echo "Building for MacOS X"
	@GOOS=darwin GOARCH=amd64 $(BUILD_FLAGS) go build -o xPipeline
	@tar -zcf dist/xPipeline_mac.tar.gz xPipeline pipeline/*.yml

freebsd:
	@echo "Building for FreeBSD"
	@GOOS=freebsd GOARCH=amd64 $(BUILD_FLAGS) go build -o xPipeline
	@tar -zcf dist/xPipeline_freebsd.tar.gz xPipeline pipeline/*.yml

win:
	@echo "Building for Windows"
	@GOOS=windows GOARCH=amd64 $(BUILD_FLAGS) go build -o xPipeline.exe
	@tar -zcf dist/xPipeline_win.tar.gz xPipeline.exe pipeline/*.yml

pi:
	@echo "Building for Raspberry Pi"
	@GOOS=linux GOARCH=arm $(BUILD_FLAGS) go build -o xPipeline
	@tar -zcf dist/xPipeline_pi.tar.gz xPipeline pipeline/*.yml

current:
	@$(BUILD_FLAGS) go build

vendor:
	@go get -u github.com/kardianos/govendor
	@govendor update +outside

test:
	@$(BUILD_FLAGS) govendor test -cover -v -timeout 10s +local

all: clean freebsd linux mac pi win aws current

.DEFAULT_GOAL := current
