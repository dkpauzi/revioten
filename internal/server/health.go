package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// healthResponse adalah isi JSON dari endpoint /health.
// Tag `json:"status"` menentukan nama field di JSON.
type healthResponse struct {
	Status string `json:"status"`
}

// handleHealth menjawab bahwa server hidup.
// Pemeriksaan database belum ada; itu menyusul di M0.3.
func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err := json.NewEncoder(w).Encode(healthResponse{Status: "ok"})
	if err != nil {
		// Status 200 sudah terkirim, jadi error ini hanya bisa dicatat.
		slog.Error("gagal menulis respons /health", "error", err)
	}
}
