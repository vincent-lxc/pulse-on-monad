package erc8004

import "testing"

func TestFromEnvDefaults(t *testing.T) {
	c := FromEnv()
	if c.IdentityRegistry != IdentityRegistryTestnet {
		t.Fatalf("registry %s", c.IdentityRegistry)
	}
	if c.ChainID != MonadTestnetChainID {
		t.Fatalf("chain %d", c.ChainID)
	}
	card := c.RegistrationFile("pulse", "demo")
	if card["type"] != "https://eips.ethereum.org/EIPS/eip-8004#registration-v1" {
		t.Fatal("bad type")
	}
	guide := c.HowToRegister()
	if guide == "" || len(guide) < 40 {
		t.Fatal("empty guide")
	}
}
