// Package approval berisi jabatan dan rute approval (M1.8, UC-02, UC-06).
// Konsep: alur kerja sebagai data. Jabatan punya key tetap dan nama yang bisa
// diganti (D-58); rute adalah daftar slot berurutan per tahap Prepared,
// Checked, dan Approved (D-45 sampai D-49, R-APR-08). Persetujuan tetap tanda
// tangan basah di kertas (D-07); paket ini hanya menyimpan siapa yang boleh
// mengisi setiap slot.
package approval

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/dkpauzi/revioten/internal/problem"
)

// Tahap tanda tangan (D-45). Urutannya tetap: Prepared, lalu Checked, lalu
// satu atau lebih Approved (D-49).
const (
	StagePrepared = "prepared"
	StageChecked  = "checked"
	StageApproved = "approved"
)

// stageNames adalah nama tahap untuk rincian audit log.
var stageNames = map[string]string{StagePrepared: "Prepared", StageChecked: "Checked", StageApproved: "Approved"}

// Key jabatan sesuai bagan organisasi NPI (D-54). Namanya hanya tampilan.
const (
	KeyPPAPEngineer    = "ppap_engineer"
	KeyPPAPStaff       = "ppap_staff"
	KeyProjectEngineer = "project_engineer"
	KeyProjectLeader   = "project_leader"
	KeyAPQPController  = "apqp_ppap_controller"
	KeyNPIManager      = "npi_manager"
	KeyQAQCManager     = "qa_qc_manager"
	KeyAdvisor         = "advisor"
	KeyFactoryManager  = "factory_manager"
)

// Position adalah satu jabatan beserta pemegangnya.
type Position struct {
	ID   string
	Key  string
	Name string
	// PerProjectGroup true untuk Project Leader dan Project Engineer: satu
	// pemegang per Project Group (D-54).
	PerProjectGroup bool
	Holders         []Holder // tidak pernah nil
}

// Holder adalah satu orang yang memegang jabatan. Project Group hanya diisi
// untuk jabatan per Project Group.
type Holder struct {
	ID               string
	PersonID         string
	PersonName       string
	ProjectGroupID   string
	ProjectGroupCode string
	CreatedAt        time.Time // selalu UTC
}

// NewHolder adalah isian pemegang jabatan baru.
type NewHolder struct {
	PersonID       string
	ProjectGroupID string
}

// PositionRef adalah jabatan yang boleh mengisi satu slot.
type PositionRef struct {
	ID   string
	Name string
}

// Slot adalah satu tempat tanda tangan di rute. PrintTitle adalah tulisan di
// bawah nama pada cetakan, terpisah dari nama jabatan (R-APR-17, D-76).
type Slot struct {
	Stage      string
	PrintTitle string
	Positions  []PositionRef
}

// Route adalah satu rute approval, dengan slot menurut urutan tanda tangan.
type Route struct {
	ID        string
	Name      string
	Slots     []Slot
	CreatedAt time.Time // selalu UTC
}

// SlotInput adalah isian satu slot.
type SlotInput struct {
	Stage       string
	PrintTitle  string
	PositionIDs []string
}

// RouteInput adalah isian rute. Saat rute diubah, Slots menggantikan semua
// slot lama.
type RouteInput struct {
	Name  string
	Slots []SlotInput
}

// Batas isian.
// ASUMSI E-30: nama paling banyak 200 karakter, seperti master lain.
// ASUMSI E-40: judul cetak paling banyak 100 karakter, dan paling banyak 10
// slot per rute; cover TR-020 punya lima slot (F-01).
const (
	maxNameLength       = 200
	maxPrintTitleLength = 100
	maxSlots            = 10
)

// cleanRoute merapikan dan memeriksa isian rute. known berisi semua jabatan
// yang ada, menurut ID. Bentuk rute mengikuti keputusan yang sudah ada: satu
// Prepared di depan, lalu satu Checked (D-48), lalu satu atau lebih Approved
// (D-46, D-49). ASUMSI E-40: rute tanpa Approved belum diizinkan (lihat E-35).
func cleanRoute(in RouteInput, known map[string]string) (RouteInput, error) {
	name, nameProblem := problem.Text("name", in.Name, true, maxNameLength)
	out := RouteInput{Name: name}
	var slotProblem *problem.Problem
	note := func(p *problem.Problem) {
		if slotProblem == nil {
			slotProblem = p
		}
	}
	switch {
	case len(in.Slots) == 0:
		note(&problem.Problem{Field: "slots", Code: problem.Required, Message: "Add the slots of the route."})
	case len(in.Slots) > maxSlots:
		note(&problem.Problem{Field: "slots", Code: problem.TooLong, Message: fmt.Sprintf("Use at most %d slots.", maxSlots)})
	case !validStages(in.Slots):
		note(&problem.Problem{Field: "slots", Code: problem.InvalidFormat,
			Message: "Use one Prepared slot, then one Checked slot, then one or more Approved slots."})
	}
	for _, s := range in.Slots {
		title, p := problem.Text("print_title", s.PrintTitle, true, maxPrintTitleLength)
		if p != nil {
			note(p)
		}
		if len(s.PositionIDs) == 0 {
			note(&problem.Problem{Field: "position_ids", Code: problem.Required, Message: "Choose at least one position for every slot."})
		}
		for i, id := range s.PositionIDs {
			if _, ok := known[id]; !ok || slices.Contains(s.PositionIDs[:i], id) {
				note(&problem.Problem{Field: "position_ids", Code: problem.InvalidFormat,
					Message: "A position does not exist or is listed twice in one slot."})
			}
		}
		out.Slots = append(out.Slots, SlotInput{Stage: s.Stage, PrintTitle: title, PositionIDs: s.PositionIDs})
	}

	var problems []problem.Problem
	for _, p := range []*problem.Problem{nameProblem, slotProblem} {
		if p != nil {
			problems = append(problems, *p)
		}
	}
	if len(problems) > 0 {
		return RouteInput{}, &problem.ValidationError{Problems: problems}
	}
	return out, nil
}

// validStages memeriksa urutan tahap: tepat satu Prepared di depan, tepat
// satu Checked sesudahnya, lalu sisanya Approved, paling sedikit satu.
func validStages(slots []SlotInput) bool {
	if len(slots) < 3 || slots[0].Stage != StagePrepared || slots[1].Stage != StageChecked {
		return false
	}
	for _, s := range slots[2:] {
		if s.Stage != StageApproved {
			return false
		}
	}
	return true
}

// describeSlots menulis slot rute untuk rincian audit log, misalnya
// "1 Prepared: Process. Eng (PPAP Engineer, PPAP Staff); 2 Checked: ...".
func describeSlots(slots []Slot) string {
	parts := make([]string, 0, len(slots))
	for i, s := range slots {
		names := make([]string, 0, len(s.Positions))
		for _, p := range s.Positions {
			names = append(names, p.Name)
		}
		parts = append(parts, fmt.Sprintf("%d %s: %s (%s)", i+1, stageNames[s.Stage], s.PrintTitle, strings.Join(names, ", ")))
	}
	return strings.Join(parts, "; ")
}
