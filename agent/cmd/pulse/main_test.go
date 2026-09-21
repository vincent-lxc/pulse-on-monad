package main

import "testing"

func TestParseArgs(t *testing.T) {
	cmd, live := parseArgs(nil)
	if cmd != "demo" || live {
		t.Fatalf("default %s %v", cmd, live)
	}
	cmd, live = parseArgs([]string{"demo"})
	if cmd != "demo" || live {
		t.Fatalf("demo %s %v", cmd, live)
	}
	cmd, live = parseArgs([]string{"demo", "--live"})
	if cmd != "demo" || !live {
		t.Fatalf("--live %s %v", cmd, live)
	}
	cmd, live = parseArgs([]string{"--live"})
	if cmd != "demo" || !live {
		t.Fatalf("bare --live %s %v", cmd, live)
	}
	cmd, live = parseArgs([]string{"version"})
	if cmd != "version" || live {
		t.Fatalf("version %s %v", cmd, live)
	}
}
