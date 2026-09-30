package main

import "testing"

func TestParseArgs(t *testing.T) {
	f := parseArgs(nil)
	if f.cmd != "demo" || f.live || f.jev {
		t.Fatalf("default %+v", f)
	}
	f = parseArgs([]string{"demo"})
	if f.cmd != "demo" || f.live || f.jev {
		t.Fatalf("demo %+v", f)
	}
	f = parseArgs([]string{"demo", "--live"})
	if f.cmd != "demo" || !f.live {
		t.Fatalf("--live %+v", f)
	}
	f = parseArgs([]string{"--live"})
	if f.cmd != "demo" || !f.live {
		t.Fatalf("bare --live %+v", f)
	}
	f = parseArgs([]string{"demo", "--jev"})
	if f.cmd != "demo" || !f.jev {
		t.Fatalf("--jev %+v", f)
	}
	f = parseArgs([]string{"demo", "--live", "--jev"})
	if !f.live || !f.jev {
		t.Fatalf("both %+v", f)
	}
	f = parseArgs([]string{"version"})
	if f.cmd != "version" || f.live {
		t.Fatalf("version %+v", f)
	}
}

func TestDemoConfigPreservesLiveEnvironment(t *testing.T) {
	for _, variable := range []string{"PULSE_STAMP_LIVE", "PULSE_LIVE_STAMP"} {
		t.Run(variable, func(t *testing.T) {
			t.Setenv("PULSE_STAMP_LIVE", "")
			t.Setenv("PULSE_LIVE_STAMP", "")
			t.Setenv(variable, "1")
			if !demoConfig(parseArgs([]string{"demo"})).Live {
				t.Fatal("CLI erased live environment")
			}
		})
	}
	t.Setenv("PULSE_STAMP_LIVE", "")
	t.Setenv("PULSE_LIVE_STAMP", "")
	if !demoConfig(parseArgs([]string{"demo", "--live"})).Live {
		t.Fatal("flag must enable live")
	}
	if demoConfig(parseArgs(nil)).Live {
		t.Fatal("default must stay dry-run")
	}
}
