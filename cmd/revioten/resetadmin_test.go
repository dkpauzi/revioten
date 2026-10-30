package main

import (
	"bytes"
	"testing"
)

// TestResetAdminFlags: -login wajib diisi, dan perintah ini tidak menerima
// password lewat flag; password sementara dibuat sistem (E-22, D-101).
func TestResetAdminFlags(t *testing.T) {
	tests := map[string][]string{
		"tanpa -login":             {},
		"flag -password tidak ada": {"-login", "admin.uji", "-password", "rahasia"},
	}
	for name, args := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := parseResetAdminFlags(args, &bytes.Buffer{}); err == nil {
				t.Fatal("ingin error")
			}
		})
	}

	opts, err := parseResetAdminFlags([]string{"-login", "admin.uji"}, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("parseResetAdminFlags: %v", err)
	}
	if opts.login != "admin.uji" || opts.envFile != ".env" {
		t.Errorf("hasil %+v", opts)
	}
}
