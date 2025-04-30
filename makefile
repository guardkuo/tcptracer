.PHONY: build

WINFLAG=GOOS=windows GOARCH=amd64
SRC=log_parse.go tcplog.go

build: clean
	@go build -o bin/tracer $(SRC) 

clean:
	@go clean

build_win: clean
	@$(WINFLAG) go build -o bin/tracer.exe $(SRC) 
