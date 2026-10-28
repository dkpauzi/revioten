package db

// Membuat kode query dari sql/queries dan skema dari sql/migrations.
// Jalankan dari root repo:
//
//	go generate ./...
//
// Versi sqlc dikunci di baris ini, jadi alat ini tidak masuk go.mod.
//go:generate go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1 generate -f ../../sqlc.yaml
