package service

import (
	"encoding/json"
	"github.com/irvanmhndra/hris-api/internal/dto"
	"testing"
)

func TestOvertimeAndAssetValidation(t *testing.T) {
	for _, body := range []string{`{"start_at":"bad","end_at":"bad"}`, `{"start_at":"2026-01-01T09:00:00+07:00","end_at":"2026-01-01T09:10:00+07:00"}`, `{"start_at":"2026-01-01T09:00:00+07:00","end_at":"2026-01-02T09:00:00+07:00"}`} {
		v := dto.HRItem{Title: "Overtime", Description: "Valid reason", Status: "pending", Data: json.RawMessage(body)}
		if ValidateHRItem("overtime", &v) == nil {
			t.Fatal("accepted invalid duration")
		}
	}
	v := dto.HRItem{Title: "Laptop", Status: "assigned", Data: json.RawMessage(`{"code":"LAP-01"}`)}
	if ValidateHRItem("assets", &v) == nil {
		t.Fatal("assigned asset without employee")
	}
}
