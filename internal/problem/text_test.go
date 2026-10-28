package problem

import (
	"strings"
	"testing"
)

func TestText(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		required bool
		want     string
		code     string // kosong berarti tidak ada masalah
	}{
		{"spasi di tepi dibuang", "  Example  ", true, "Example", ""},
		{"wajib tapi kosong", "   ", true, "", Required},
		{"tidak wajib dan kosong", "", false, "", ""},
		{"tepat 10 karakter", strings.Repeat("é", 10), true, strings.Repeat("é", 10), ""},
		{"lebih dari 10 karakter", strings.Repeat("a", 11), true, "", TooLong},
		{"baris baru", "satu\ndua", true, "", InvalidFormat},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, p := Text("name", tt.value, tt.required, 10)
			if tt.code == "" {
				if p != nil || got != tt.want {
					t.Errorf("Text = %q, %+v; ingin %q tanpa masalah", got, p, tt.want)
				}
				return
			}
			if p == nil || p.Code != tt.code || p.Field != "name" || p.Message == "" {
				t.Errorf("masalah %+v, ingin kode %s untuk field name", p, tt.code)
			}
		})
	}
}
