.PHONY: fmt install lint

clean:
	go clean ./...
fmt:
	gofmt -l -w -tabs=true $(GOPATH)/src/github.com/zimmski/tavor
install:
	go install ./...
lint: clean
	go tool vet -all=true -v=true $(GOPATH)/src/github.com/zimmski/tavor
	golint $(GOPATH)/src/github.com/zimmski/tavor
test: clean
	go test ./...