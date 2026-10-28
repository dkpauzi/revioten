package server

import (
	"context"
	"time"
)

// healthTimeout adalah batas waktu menunggu jawaban database di /health.
const healthTimeout = 2 * time.Second

// GetHealth menjawab apakah server dan database bisa dipakai. Bila database
// tidak bisa dihubungi, jawabannya 503 dan detailnya hanya dicatat di log.
func (a *apiServer) GetHealth(ctx context.Context, _ GetHealthRequestObject) (GetHealthResponseObject, error) {
	pingCtx, cancel := context.WithTimeout(ctx, healthTimeout)
	defer cancel()

	if err := a.db.Ping(pingCtx); err != nil {
		a.log.Warn("database tidak bisa dihubungi", "request_id", requestIDFrom(ctx), "error", err)
		return GetHealth503JSONResponse{Status: HealthStatusUnavailable, Database: HealthDatabaseUnreachable}, nil
	}
	return GetHealth200JSONResponse{Status: HealthStatusOk, Database: HealthDatabaseOk}, nil
}
