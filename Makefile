build:
	GOOS=darwin GOARCH=amd64 go build -v .
	GOOS=darwin GOARCH=arm64 go build -v .
	GOOS=linux GOARCH=amd64 go build -v .
	GOOS=linux GOARCH=arm64 go build -v .
	GOOS=windows GOARCH=amd64 go build -v .
	GOOS=windows GOARCH=arm64 go build -v .

lint:
	GOOS=linux golangci-lint run .
	GOOS=windows golangci-lint run .
	GOOS=darwin golangci-lint run .

lint_install:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

test:
	go test -v ./...

build_bin:
	GOOS=darwin GOARCH=amd64 go build -o bin/example_darwin_amd64 example/main.go
	GOOS=darwin GOARCH=arm64 go build -o bin/example_darwin_arm64 example/main.go
	GOOS=linux GOARCH=amd64 go build -o bin/example_linux_amd64 example/main.go
	GOOS=linux GOARCH=arm64 go build -o bin/example_linux_arm64 example/main.go
	GOOS=windows GOARCH=amd64 go build -o bin/example_windows_amd64 example/main.go
	GOOS=windows GOARCH=arm64 go build -o bin/example_windows_arm64 example/main.go