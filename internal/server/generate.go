package server

// Membuat api.gen.go dari kontrak API. Jalankan dari root repo:
//
//	go generate ./...
//
// Versi oapi-codegen dikunci di baris ini, jadi alat ini tidak masuk go.mod.
//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.8.1 -config oapi-codegen.yaml ../../api/openapi.yaml
