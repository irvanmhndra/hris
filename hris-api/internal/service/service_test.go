package service

import (
	"github.com/irvanmhndra/hris-api/internal/dto"
	"testing"
)

func TestLeaveValidation(t *testing.T) {
	for _, tc := range []struct {
		name  string
		input dto.Leave
		valid bool
	}{
		{"valid", dto.Leave{Kind: "annual", StartDate: "2026-11-01", EndDate: "2026-11-02", Reason: "Family trip"}, true},
		{"reversed", dto.Leave{Kind: "annual", StartDate: "2026-11-02", EndDate: "2026-11-01", Reason: "Family trip"}, false},
		{"invalid calendar date", dto.Leave{Kind: "annual", StartDate: "2026-02-30", EndDate: "2026-03-01", Reason: "Family trip"}, false},
		{"invalid kind", dto.Leave{Kind: "other", StartDate: "2026-11-01", EndDate: "2026-11-02", Reason: "Family trip"}, false},
		{"blank reason", dto.Leave{Kind: "annual", StartDate: "2026-11-01", EndDate: "2026-11-02", Reason: "     "}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if (ValidateLeave(tc.input) == nil) != tc.valid {
				t.Fatal("unexpected validation result")
			}
		})
	}
}
