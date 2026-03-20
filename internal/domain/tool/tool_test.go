package tool_test

import (
	"testing"

	domain "tool_management_backend/internal/domain/tool"
)

func TestPrepareForUpdateAssignedAppendsHistory(t *testing.T) {
	current := domain.Tool{
		ID:     "507f1f77bcf86cd799439011",
		Status: domain.StatusAssigned,
		AssignmentHistory: []domain.AssignmentHistory{
			{Responsible: "Ana", AssignmentDate: "2025-01-01"},
		},
	}

	updated, err := domain.PrepareForUpdate(current, domain.Tool{
		Code:           "T-1",
		Name:           "Taladro",
		Type:           domain.TypeElectric,
		Status:         domain.StatusAssigned,
		Responsible:    "Luis",
		AssignmentDate: "2025-02-01",
		Location:       "Almacen",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(updated.AssignmentHistory) != 2 {
		t.Fatalf("expected history length 2, got %d", len(updated.AssignmentHistory))
	}
}

func TestPrepareForUpdateMaintenanceAppendsHistory(t *testing.T) {
	current := domain.Tool{ID: "507f1f77bcf86cd799439011"}

	updated, err := domain.PrepareForUpdate(current, domain.Tool{
		Code:            "T-1",
		Name:            "Taladro",
		Type:            domain.TypeElectric,
		Status:          domain.StatusMaintenance,
		DateMaintenance: "2025-02-01",
		NextMaintenance: "2025-03-01",
		Location:        "Almacen",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(updated.MaintenanceRecord) != 1 {
		t.Fatalf("expected maintenance history length 1, got %d", len(updated.MaintenanceRecord))
	}
}

func TestValidateRequiresAssignedFields(t *testing.T) {
	err := domain.Validate(domain.Normalize(domain.Tool{
		Code:     "T-1",
		Name:     "Taladro",
		Type:     domain.TypeElectric,
		Status:   domain.StatusAssigned,
		Location: "Almacen",
	}))
	if err == nil {
		t.Fatal("expected validation error")
	}

	if _, ok := err.(domain.ValidationError); !ok {
		t.Fatalf("expected validation error type, got %T", err)
	}
}
