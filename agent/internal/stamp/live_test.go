package stamp

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestWantLiveFromEnv(t *testing.T) {
	t.Setenv("PULSE_STAMP_LIVE", "")
	t.Setenv("PULSE_LIVE_STAMP", "")
	if WantLiveFromEnv() {
		t.Fatal("empty env should be dry-run")
	}
	t.Setenv("PULSE_STAMP_LIVE", "1")
	if !WantLiveFromEnv() {
		t.Fatal("PULSE_STAMP_LIVE=1")
	}
	t.Setenv("PULSE_STAMP_LIVE", "")
	t.Setenv("PULSE_LIVE_STAMP", "true")
	if !WantLiveFromEnv() {
		t.Fatal("legacy PULSE_LIVE_STAMP")
	}
}

func TestExplorerTxURL(t *testing.T) {
	u := ExplorerTxURL("0xabc")
	if u != "https://testnet.monadvision.com/tx/0xabc" {
		t.Fatal(u)
	}
	if ExplorerTxURL("") != "" {
		t.Fatal("empty")
	}
	if ExplorerTxURL("dead") != "https://testnet.monadvision.com/tx/0xdead" {
		t.Fatal(ExplorerTxURL("dead"))
	}
}

func TestParseStampedID(t *testing.T) {
	id := big.NewInt(42)
	lg := &types.Log{
		Topics: []common.Hash{
			StampedTopic0(),
			common.BigToHash(id),
			crypto.Keccak256Hash([]byte("hash")),
			common.BigToHash(big.NewInt(1)),
		},
	}
	got, err := ParseStampedID([]*types.Log{lg})
	if err != nil {
		t.Fatal(err)
	}
	if got.Cmp(id) != 0 {
		t.Fatalf("id %s", got)
	}
	if _, err := ParseStampedID(nil); err == nil {
		t.Fatal("expected missing event")
	}
}

func TestEncodeGetReceipt(t *testing.T) {
	data, err := EncodeGetReceipt(big.NewInt(7))
	if err != nil {
		t.Fatal(err)
	}
	sel := crypto.Keccak256([]byte("getReceipt(uint256)"))[:4]
	if common.Bytes2Hex(data[:4]) != common.Bytes2Hex(sel) {
		t.Fatalf("selector %x want %x", data[:4], sel)
	}
	if data[len(data)-1] != 7 {
		t.Fatalf("id word %x", data[4:])
	}
}

func TestLiveStampRequiresKey(t *testing.T) {
	var h [32]byte
	h[0] = 1
	cfg := Config{RPC: DefaultRPC, ChainID: DefaultChainID, Contract: common.HexToAddress(DefaultContract), DryRun: false}
	_, err := cfg.Stamp(context.Background(), Request{DecisionHash: h, Action: 1, Note: "agent=test"})
	if err == nil || !contains(err.Error(), "PRIVATE_KEY") {
		t.Fatalf("expected PRIVATE_KEY error, got %v", err)
	}
}

func TestStampedTopic0Stable(t *testing.T) {
	a := StampedTopic0()
	b := crypto.Keccak256Hash([]byte(StampedEvent))
	if a != b {
		t.Fatal(a.Hex())
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		func() bool {
			for i := 0; i+len(sub) <= len(s); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		}())
}
