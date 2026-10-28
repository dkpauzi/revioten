package customer

import (
	"slices"
	"strings"
	"testing"
)

// TestValidate memeriksa aturan input customer tanpa database.
func TestValidate(t *testing.T) {
	tests := []struct {
		name string
		in   NewCustomer
		want NewCustomer // hasil setelah dirapikan, bila input diterima
		// problems berisi "field:kode" yang diharapkan, urut seperti di hasil.
		problems []string
	}{
		{
			name: "kode dan nama benar",
			in:   NewCustomer{Code: "AB", Name: "Example Customer", ProjectGroupID: "grup-1"},
			want: NewCustomer{Code: "AB", Name: "Example Customer", ProjectGroupID: "grup-1"},
		},
		{
			name: "spasi di tepi dibuang dan huruf dibuat besar",
			in:   NewCustomer{Code: " ab ", Name: "  Example  Customer \t", ProjectGroupID: "grup-1"},
			want: NewCustomer{Code: "AB", Name: "Example  Customer", ProjectGroupID: "grup-1"},
		},
		{
			name: "nama 200 karakter boleh, dihitung per karakter bukan per byte",
			in:   NewCustomer{Code: "AB", Name: strings.Repeat("é", 200), ProjectGroupID: "grup-1"},
			want: NewCustomer{Code: "AB", Name: strings.Repeat("é", 200), ProjectGroupID: "grup-1"},
		},
		{name: "kode kosong", in: NewCustomer{Code: "  ", Name: "X", ProjectGroupID: "grup-1"}, problems: []string{"code:required"}},
		{name: "kode satu huruf", in: NewCustomer{Code: "A", Name: "X", ProjectGroupID: "grup-1"}, problems: []string{"code:invalid_format"}},
		{name: "kode tiga huruf", in: NewCustomer{Code: "ABC", Name: "X", ProjectGroupID: "grup-1"}, problems: []string{"code:invalid_format"}},
		{name: "kode berisi angka", in: NewCustomer{Code: "A1", Name: "X", ProjectGroupID: "grup-1"}, problems: []string{"code:invalid_format"}},
		{name: "kode huruf di luar A sampai Z", in: NewCustomer{Code: "äb", Name: "X", ProjectGroupID: "grup-1"}, problems: []string{"code:invalid_format"}},
		{name: "kode ZZ dicadangkan", in: NewCustomer{Code: "zz", Name: "X", ProjectGroupID: "grup-1"}, problems: []string{"code:reserved"}},
		{name: "nama kosong", in: NewCustomer{Code: "AB", Name: " \t ", ProjectGroupID: "grup-1"}, problems: []string{"name:required"}},
		{name: "nama 201 karakter", in: NewCustomer{Code: "AB", Name: strings.Repeat("a", 201), ProjectGroupID: "grup-1"}, problems: []string{"name:too_long"}},
		{name: "nama berisi baris baru", in: NewCustomer{Code: "AB", Name: "Example\nCustomer", ProjectGroupID: "grup-1"}, problems: []string{"name:invalid_format"}},
		{name: "semua kosong", in: NewCustomer{}, problems: []string{"code:required", "name:required", "project_group_id:required"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, problems := validate(tt.in)

			if len(tt.problems) == 0 {
				if len(problems) > 0 {
					t.Fatalf("input benar ditolak: %+v", problems)
				}
				if got != tt.want {
					t.Errorf("hasil = %+v, ingin %+v", got, tt.want)
				}
				return
			}

			var codes []string
			for _, p := range problems {
				codes = append(codes, p.Field+":"+p.Code)
				if p.Message == "" {
					t.Errorf("masalah %s:%s tanpa pesan", p.Field, p.Code)
				}
			}
			if !slices.Equal(codes, tt.problems) {
				t.Errorf("masalah = %v, ingin %v", codes, tt.problems)
			}
		})
	}
}
