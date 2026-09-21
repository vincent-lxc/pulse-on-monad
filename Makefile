.PHONY: test demo demo-live forge go-test

test: forge go-test

forge:
	cd contracts && forge test

go-test:
	cd agent && go test ./...

demo:
	cd agent && go run ./cmd/pulse demo

# Broadcasts stamp() — needs PRIVATE_KEY + testnet MON. Do not use in CI.
demo-live:
	cd agent && go run ./cmd/pulse demo --live
