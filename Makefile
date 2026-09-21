.PHONY: test demo demo-live forge go-test receipts receipts-test

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

# Read-only receipts page. Open http://127.0.0.1:8080
receipts:
	cd web && python3 -m http.server 8080

receipts-test:
	cd web && node --test receipts.test.mjs
