package server

import (
	"context"

	"github.com/dkpauzi/revioten/internal/problem"
	"github.com/dkpauzi/revioten/internal/process"
)

// ListProcesses menjawab GET /api/v1/processes.
func (a *apiServer) ListProcesses(ctx context.Context, _ ListProcessesRequestObject) (ListProcessesResponseObject, error) {
	list, err := a.processes.List(ctx, actorFrom(ctx))
	if err != nil {
		return nil, err
	}
	items := make([]Process, 0, len(list))
	for _, p := range list {
		items = append(items, toAPIProcess(p))
	}
	return ListProcesses200JSONResponse{Items: items}, nil
}

// GetProcess menjawab GET /api/v1/processes/{processId}.
func (a *apiServer) GetProcess(ctx context.Context, req GetProcessRequestObject) (GetProcessResponseObject, error) {
	p, err := a.processes.Get(ctx, actorFrom(ctx), req.ProcessId)
	if err != nil {
		return nil, err
	}
	return GetProcess200JSONResponse(toAPIProcess(p)), nil
}

// CreateProcess menjawab POST /api/v1/processes.
func (a *apiServer) CreateProcess(ctx context.Context, req CreateProcessRequestObject) (CreateProcessResponseObject, error) {
	in := process.Input{Name: req.Body.Name}
	if req.Body.Aliases != nil {
		in.Aliases = *req.Body.Aliases
	}
	created, err := a.processes.Create(ctx, actorFrom(ctx), in)
	if err != nil {
		return nil, err
	}
	return CreateProcess201JSONResponse(toAPIProcess(created)), nil
}

// UpdateProcess menjawab PUT /api/v1/processes/{processId}.
func (a *apiServer) UpdateProcess(ctx context.Context, req UpdateProcessRequestObject) (UpdateProcessResponseObject, error) {
	// aliases wajib di kontrak, karena isinya menggantikan semua alias. JSON []
	// menjadi daftar kosong, sedangkan field yang tidak dikirim menjadi nil:
	// klien yang lupa field ini ditolak, bukan menghapus semua alias.
	// ASUMSI E-34: kode hasil generate tidak memeriksa field wajib, jadi
	// field yang nilai kosongnya berbahaya diperiksa di sini.
	if req.Body.Aliases == nil {
		return nil, &problem.ValidationError{Problems: []problem.Problem{{
			Field: "aliases", Code: problem.Required, Message: "Send the full list of aliases, or [] for none.",
		}}}
	}
	in := process.Input{Name: req.Body.Name, Aliases: req.Body.Aliases}
	updated, err := a.processes.Update(ctx, actorFrom(ctx), req.ProcessId, in)
	if err != nil {
		return nil, err
	}
	return UpdateProcess200JSONResponse(toAPIProcess(updated)), nil
}

// toAPIProcess mengubah Process domain menjadi Process kontrak API.
func toAPIProcess(p process.Process) Process {
	return Process{Id: p.ID, Name: p.Name, Aliases: p.Aliases, CreatedAt: p.CreatedAt}
}
