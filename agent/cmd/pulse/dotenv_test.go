package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDotEnvDoesNotOverride(t *testing.T) {
	dir := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
	if err := os.WriteFile(".env", []byte("PULSE_AGENT_ID=from-file\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PULSE_AGENT_ID", "from-env")
	loadDotEnv()
	if os.Getenv("PULSE_AGENT_ID") != "from-env" {
		t.Fatalf("overrode process env: %s", os.Getenv("PULSE_AGENT_ID"))
	}
}

func TestLoadDotEnvSetsMissing(t *testing.T) {
	dir := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
	key := "PULSE_DOTENV_TEST_" + filepath.Base(dir)
	_ = os.Unsetenv(key)
	if err := os.WriteFile(".env", []byte(key+"=hello\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	loadDotEnv()
	if os.Getenv(key) != "hello" {
		t.Fatalf("got %q", os.Getenv(key))
	}
	_ = os.Unsetenv(key)
}
