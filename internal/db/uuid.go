package db

import "regexp"

// uuidPattern mengenali bentuk UUID, misalnya 01a180aa-a98b-7bf2-8e0d-b97a25ecc489.
var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// IsUUID melaporkan apakah s berbentuk UUID. ID dari klien diperiksa dulu
// dengan fungsi ini, supaya teks sembarang berarti "tidak ada" (404), bukan
// error database (500).
func IsUUID(s string) bool {
	return uuidPattern.MatchString(s)
}
