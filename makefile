.PHONY: build
WINFLAG=GOOS=windows GOARCH=amd64
SRC = tcplog.go log_parse.go


build: clean
	@go build -o bin/tcptracer.exe $(SRC)

clean:
	@go clean

build_win: clean
	@$(WINFLAG) go build -o bin/tcptracer.exe $(SRC)

format-code:
	@go fmt $(SRC)
