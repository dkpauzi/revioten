package template

import (
	"slices"
	"testing"
)

// Bagian C (E-144, migrasi 00056): field header bawaan punya key tetap,
// PFMEA v1 dan Control Plan v1 punya field Remark, dan judul grup kolom
// mengikuti SF-TR-020-03 dan SC-TR-020-04.
func TestPrintHeadFields(t *testing.T) {
	f := newFixture(t)
	keys := func(v FormatVersion, area string) []string {
		var out []string
		for _, fl := range v.Fields {
			if fl.Area == area {
				out = append(out, fl.Key+":"+fl.Label)
			}
		}
		return out
	}
	if got := keys(f.version(t, "PFMEA v1"), AreaHeader); !slices.Equal(got, []string{
		"process_line:Process or Line", "process_responsibility:Process Responsibility", "key_date:Key Date", "core_team:Core Team", "remark:Remark"}) {
		t.Errorf("header PFMEA v1: %v", got)
	}
	if got := keys(f.version(t, "Control Plan v1"), AreaHeader); !slices.Equal(got, []string{"process_line:Process or Line", "core_team:Core Team", "remark:Remark"}) {
		t.Errorf("header Control Plan v1: %v", got)
	}
	if got := keys(f.version(t, "PFD v1"), AreaHeader); !slices.Equal(got, []string{"process_line:Process or Line"}) {
		t.Errorf("header PFD v1: %v", got)
	}
	groups := func(v FormatVersion) map[string]string {
		out := map[string]string{}
		for _, fl := range v.Fields {
			if fl.Area == AreaRow {
				out[fl.Label] = fl.Group
			}
		}
		return out
	}
	cp := groups(f.version(t, "Control Plan v1"))
	for label, want := range map[string]string{
		"No": "PROCESS", "Process Name / Operation Description": "PROCESS", "Product Characteristics": "CHARACTERISTICS ( Key Parameter )",
		"Process Specification / Tolerance": "CHARACTERISTICS ( Key Parameter )", "Method": "CONTROL METHOD", "Relevant Document": "CONTROL METHOD",
		"Process Flow": "", "Special Characteristics": "", "Reaction Plan": "",
	} {
		if cp[label] != want {
			t.Errorf("grup CP %q: %q, ingin %q", label, cp[label], want)
		}
	}
	pfmea := groups(f.version(t, "PFMEA v1"))
	for label, want := range map[string]string{
		"Current Process Controls Prevention": "Current Process", "Detection": "Current Process", "Actions Taken & Completion Date": "Action Results",
		"Result RPN": "Action Results", "RPN": "", "Severity": "",
	} {
		if pfmea[label] != want {
			t.Errorf("grup PFMEA %q: %q, ingin %q", label, pfmea[label], want)
		}
	}
}
