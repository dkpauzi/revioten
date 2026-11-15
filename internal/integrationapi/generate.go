// Package integrationapi adalah adapter HTTP untuk API integrasi
// /integration/v1 (Tahap 9). Aturannya ada di internal/integration; paket ini
// hanya menerjemahkan HTTP ke panggilan layanan dan sebaliknya (R-API-01).
package integrationapi

// Membuat api.gen.go dari kontrak integrasi. Jalankan dari root repo:
//
//	go generate ./...
//
//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.8.1 -config oapi-codegen.yaml ../../api/integration.yaml
