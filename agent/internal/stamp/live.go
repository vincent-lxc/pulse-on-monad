package stamp

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

const (
	ExplorerTxBase = "https://testnet.monadvision.com/tx/"
	StampedEvent   = "Stamped(uint256,bytes32,uint256,uint256,uint8,address,uint64,string)"
)

const contractABIJSON = `[
  {"type":"function","name":"stamp","stateMutability":"nonpayable","inputs":[{"name":"decisionHash","type":"bytes32"},{"name":"symbolId","type":"uint256"},{"name":"sizeHint","type":"uint256"},{"name":"action","type":"uint8"},{"name":"note","type":"string"}],"outputs":[{"name":"id","type":"uint256"}]},
  {"type":"function","name":"getReceipt","stateMutability":"view","inputs":[{"name":"id","type":"uint256"}],"outputs":[{"name":"","type":"tuple","components":[
    {"name":"decisionHash","type":"bytes32"},
    {"name":"symbolId","type":"uint256"},
    {"name":"sizeHint","type":"uint256"},
    {"name":"action","type":"uint8"},
    {"name":"note","type":"string"},
    {"name":"stampedAt","type":"uint64"},
    {"name":"stamper","type":"address"}
  ]}]},
  {"type":"event","name":"Stamped","inputs":[
    {"name":"id","type":"uint256","indexed":true},
    {"name":"decisionHash","type":"bytes32","indexed":true},
    {"name":"symbolId","type":"uint256","indexed":true},
    {"name":"sizeHint","type":"uint256","indexed":false},
    {"name":"action","type":"uint8","indexed":false},
    {"name":"stamper","type":"address","indexed":false},
    {"name":"stampedAt","type":"uint64","indexed":false},
    {"name":"note","type":"string","indexed":false}
  ]}
]`

// OnchainReceipt is PulseTradeStamp.getReceipt.
type OnchainReceipt struct {
	DecisionHash [32]byte
	SymbolID     *big.Int
	SizeHint     *big.Int
	Action       uint8
	Note         string
	StampedAt    uint64
	Stamper      common.Address
}

func contractABI() abi.ABI {
	parsed, err := abi.JSON(strings.NewReader(contractABIJSON))
	if err != nil {
		panic(err)
	}
	return parsed
}

// WantLiveFromEnv is true when PULSE_STAMP_LIVE (or legacy PULSE_LIVE_STAMP) is set.
func WantLiveFromEnv() bool {
	return envTruthy("PULSE_STAMP_LIVE") || envTruthy("PULSE_LIVE_STAMP")
}

func envTruthy(key string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(key))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// ExplorerTxURL is the MonadVision testnet tx page.
func ExplorerTxURL(txHash string) string {
	h := strings.TrimSpace(txHash)
	if h == "" {
		return ""
	}
	if !strings.HasPrefix(h, "0x") && !strings.HasPrefix(h, "0X") {
		h = "0x" + h
	}
	return ExplorerTxBase + h
}

// StampedTopic0 is keccak256 of the Stamped event signature.
func StampedTopic0() common.Hash {
	return crypto.Keccak256Hash([]byte(StampedEvent))
}

// ParseStampedID reads the indexed receipt id from a Stamped log.
func ParseStampedID(logs []*types.Log) (*big.Int, error) {
	want := StampedTopic0()
	for _, lg := range logs {
		if lg == nil || len(lg.Topics) < 2 {
			continue
		}
		if lg.Topics[0] != want {
			continue
		}
		return new(big.Int).SetBytes(lg.Topics[1].Bytes()), nil
	}
	return nil, fmt.Errorf("stamp: Stamped event not found")
}

// EncodeGetReceipt ABI-encodes getReceipt(uint256).
func EncodeGetReceipt(id *big.Int) ([]byte, error) {
	if id == nil {
		return nil, fmt.Errorf("stamp: nil receipt id")
	}
	return contractABI().Pack("getReceipt", id)
}

func waitReceipt(ctx context.Context, client *ethclient.Client, hash common.Hash) (*types.Receipt, error) {
	deadline := time.Now().Add(45 * time.Second)
	tick := time.NewTicker(400 * time.Millisecond)
	defer tick.Stop()
	for {
		rec, err := client.TransactionReceipt(ctx, hash)
		if err == nil && rec != nil {
			return rec, nil
		}
		if err != nil && err != ethereum.NotFound {
			return nil, err
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("stamp: timed out waiting for %s", hash.Hex())
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-tick.C:
		}
	}
}

func callGetReceipt(ctx context.Context, client *ethclient.Client, to common.Address, id *big.Int) (*OnchainReceipt, error) {
	data, err := EncodeGetReceipt(id)
	if err != nil {
		return nil, err
	}
	raw, err := client.CallContract(ctx, ethereum.CallMsg{To: &to, Data: data}, nil)
	if err != nil {
		return nil, err
	}
	return unpackReceipt(raw)
}

func unpackReceipt(raw []byte) (*OnchainReceipt, error) {
	type tuple struct {
		DecisionHash [32]byte
		SymbolID     *big.Int
		SizeHint     *big.Int
		Action       uint8
		Note         string
		StampedAt    uint64
		Stamper      common.Address
	}
	var t tuple
	if err := contractABI().UnpackIntoInterface(&t, "getReceipt", raw); err != nil {
		return nil, fmt.Errorf("stamp: unpack getReceipt: %w", err)
	}
	return &OnchainReceipt{
		DecisionHash: t.DecisionHash,
		SymbolID:     t.SymbolID,
		SizeHint:     t.SizeHint,
		Action:       t.Action,
		Note:         t.Note,
		StampedAt:    t.StampedAt,
		Stamper:      t.Stamper,
	}, nil
}
