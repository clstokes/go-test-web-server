BINARY_NAME=go-test-web-server

default:
	go build -o bin/$(BINARY_NAME)

publish:
	GOOS=linux GOARCH=amd64 go build -o bin/$(BINARY_NAME)
	md5 bin/$(BINARY_NAME)
	aws s3 cp bin/$(BINARY_NAME) s3://hashicorp-cameron-public/projects/$(BINARY_NAME)/bin/$(BINARY_NAME)

fmt:
	gofmt -w .

.PHONY: default
