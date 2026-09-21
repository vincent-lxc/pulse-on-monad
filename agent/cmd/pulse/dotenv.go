package main

import (
	"bufio"
	"os"
	"strings"
)

// loadDotEnv reads a local .env if present. Existing process env wins.
// Never prints values (PRIVATE_KEY may be in the file).
func loadDotEnv() {
	for _, p := range []string{".env", "../.env"} {
		f, err := os.Open(p)
		if err != nil {
			continue
		}
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			k, v, ok := strings.Cut(line, "=")
			if !ok {
				continue
			}
			k = strings.TrimSpace(k)
			v = strings.TrimSpace(v)
			v = strings.Trim(v, `"'`)
			if k == "" {
				continue
			}
			if _, set := os.LookupEnv(k); set {
				continue
			}
			_ = os.Setenv(k, v)
		}
		_ = f.Close()
		return
	}
}
