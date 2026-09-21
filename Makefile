.PHONY: test demo forge go-test

test: forge go-test

forge:
	cd contracts && forge test

go-test:
	cd agent && go test ./...

demo:
	cd agent && go run ./cmd/pulse demo
