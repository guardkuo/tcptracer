.PHONY: build
SRC = log_parse.go tcplog.go

build: clean
	@go build -o bin/tcptracer.exe $(SRC)

clean:
	@go clean

build_win: clean
	@set GOOS=windows
	@set GOARCH=amd64
	@go build -o bin/tcptracer.exe $(SRC)

format-code:
	@go fmt $(SRC)
