package erc8004

import (
	"fmt"
	"os"
	"strings"
)

// Monad testnet Identity Registry (ERC-8004 Trustless Agents).
// Same singleton address as other public testnets in the 8004 deployment set.
const (
	MonadTestnetChainID      = 10143
	IdentityRegistryTestnet  = "0x8004A818BFB912233c491871b3d84c89A494BD9e"
	ReputationRegistryTestnet = "0x8004B663056A597Dffe9eCcC1965A193B7388713"
	AgentRegistryID          = "eip155:10143:0x8004A818BFB912233c491871b3d84c89A494BD9e"
)

// Config is a thin identity shell. Live mint is optional and never required for CI.
type Config struct {
	ChainID          int64  `json:"chain_id"`
	IdentityRegistry string `json:"identity_registry"`
	AgentURI         string `json:"agent_uri"`
	AgentWallet      string `json:"agent_wallet"`
	AgentID          string `json:"agent_id,omitempty"` // ERC-721 tokenId after register()
}

func FromEnv() Config {
	return Config{
		ChainID:          MonadTestnetChainID,
		IdentityRegistry: getenv("ERC8004_IDENTITY_REGISTRY", IdentityRegistryTestnet),
		AgentURI:         os.Getenv("ERC8004_AGENT_URI"),
		AgentWallet:      os.Getenv("ERC8004_AGENT_WALLET"),
		AgentID:          os.Getenv("ERC8004_AGENT_ID"),
	}
}

func getenv(k, def string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return def
}

// RegistrationFile is the ERC-8004 agent card stub (not uploaded in CI).
func (c Config) RegistrationFile(name, description string) map[string]any {
	regs := []map[string]any{}
	if c.AgentID != "" {
		regs = append(regs, map[string]any{
			"agentId":       c.AgentID,
			"agentRegistry": AgentRegistryID,
		})
	}
	services := []map[string]any{}
	if c.AgentWallet != "" {
		services = append(services, map[string]any{
			"name":     "wallet",
			"endpoint": c.AgentWallet,
		})
	}
	return map[string]any{
		"type":        "https://eips.ethereum.org/EIPS/eip-8004#registration-v1",
		"name":        name,
		"description": description,
		"image":       "",
		"services":    services,
		"x402Support": false,
		"active":      true,
		"registrations": regs,
		"supportedTrust": []string{"reputation"},
	}
}

// HowToRegister is human-readable; used by the demo and docs. No RPC call.
func (c Config) HowToRegister() string {
	uri := c.AgentURI
	if uri == "" {
		uri = "ipfs://<cid-of-agent-card.json>"
	}
	return fmt.Sprintf(`ERC-8004 identity (optional; not required for CI or the stamp demo)

Identity Registry (Monad testnet, chainId %d):
  %s
Explorer: https://testnet.monadvision.com/address/%s

Register an agentId (ERC-721 tokenId) from a funded wallet:

  cast send %s "register(string)" %q --rpc-url $MONAD_RPC_URL --private-key $PRIVATE_KEY

Or register() with no URI, then setAgentURI(uint256,string) after uploading the card.

agentURI:  %s
agentWallet: %s
agentId: %s

Global id once minted: eip155:%d:%s / <agentId>
`, c.ChainID, c.IdentityRegistry, c.IdentityRegistry, c.IdentityRegistry, uri, uri, emptyDash(c.AgentWallet), emptyDash(c.AgentID), c.ChainID, c.IdentityRegistry)
}

func emptyDash(s string) string {
	if s == "" {
		return "(unset — stub only)"
	}
	return s
}
