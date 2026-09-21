package stamp

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

const (
	DefaultRPC      = "https://testnet-rpc.monad.xyz"
	DefaultChainID  = 10143
	DefaultContract = "0x6eC692C5792AD122c459AC8Df3FDF67eE80842F4"
)

// Config is filled from env. Dry-run is the default / CI path.
type Config struct {
	RPC        string
	ChainID    int64
	Contract   common.Address
	PrivateKey *ecdsa.PrivateKey
	DryRun     bool
}

type Result struct {
	Calldata string `json:"calldata"`
	To       string `json:"to"`
	DryRun   bool   `json:"dry_run"`
	TxHash   string `json:"tx_hash,omitempty"`
}

func FromEnv() (Config, error) {
	rpc := getenv("MONAD_RPC_URL", DefaultRPC)
	var chain int64 = DefaultChainID
	if v := os.Getenv("MONAD_CHAIN_ID"); v != "" {
		var n int64
		if _, err := fmt.Sscanf(v, "%d", &n); err == nil && n > 0 {
			chain = n
		}
	}
	addr := getenv("PULSE_TRADE_STAMP", DefaultContract)
	if !common.IsHexAddress(addr) {
		return Config{}, fmt.Errorf("stamp: invalid PULSE_TRADE_STAMP %q", addr)
	}
	dry := true
	if v := strings.ToLower(os.Getenv("PULSE_DRY_RUN")); v == "false" || v == "0" || v == "no" {
		dry = false
	}
	cfg := Config{
		RPC:      rpc,
		ChainID:  chain,
		Contract: common.HexToAddress(addr),
		DryRun:   dry,
	}
	if pk := strings.TrimSpace(os.Getenv("PRIVATE_KEY")); pk != "" {
		key, err := crypto.HexToECDSA(strings.TrimPrefix(pk, "0x"))
		if err != nil {
			return Config{}, fmt.Errorf("stamp: PRIVATE_KEY: %w", err)
		}
		cfg.PrivateKey = key
	}
	return cfg, nil
}

func getenv(k, def string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return def
}

// Stamp encodes calldata and either returns it (dry-run) or sends the tx.
func (c Config) Stamp(ctx context.Context, req Request) (*Result, error) {
	data, err := EncodeCalldata(req)
	if err != nil {
		return nil, err
	}
	out := &Result{
		Calldata: Hex(data),
		To:       c.Contract.Hex(),
		DryRun:   c.DryRun,
	}
	if c.DryRun {
		return out, nil
	}
	if c.PrivateKey == nil {
		return nil, fmt.Errorf("stamp: live send requires PRIVATE_KEY (or keep PULSE_DRY_RUN=true)")
	}
	client, err := ethclient.DialContext(ctx, c.RPC)
	if err != nil {
		return nil, fmt.Errorf("stamp: rpc: %w", err)
	}
	defer client.Close()

	from := crypto.PubkeyToAddress(c.PrivateKey.PublicKey)
	nonce, err := client.PendingNonceAt(ctx, from)
	if err != nil {
		return nil, err
	}
	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, err
	}
	gas := uint64(200_000)
	tx := types.NewTransaction(nonce, c.Contract, big.NewInt(0), gas, gasPrice, data)
	signer := types.LatestSignerForChainID(big.NewInt(c.ChainID))
	signed, err := types.SignTx(tx, signer, c.PrivateKey)
	if err != nil {
		return nil, err
	}
	if err := client.SendTransaction(ctx, signed); err != nil {
		return nil, fmt.Errorf("stamp: send: %w", err)
	}
	out.TxHash = signed.Hash().Hex()
	return out, nil
}
