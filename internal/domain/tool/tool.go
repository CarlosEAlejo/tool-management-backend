package tool

import (
	"errors"
	"strings"
)

const (
	StatusActive      = "active"
	StatusAssigned    = "assigned"
	StatusMaintenance = "maintenance"
	StatusLost        = "lost"
	StatusDamaged     = "damaged"
)

const (
	TypeElectric  = "electric"
	TypeManual    = "manual"
	TypeMeasuring = "measuring"
	TypeSafety    = "safety"
	TypeOther     = "other"
)

var validStatuses = map[string]struct{}{
	StatusActive:      {},
	StatusAssigned:    {},
	StatusMaintenance: {},
	StatusLost:        {},
	StatusDamaged:     {},
}

var validTypes = map[string]struct{}{
	TypeElectric:  {},
	TypeManual:    {},
	TypeMeasuring: {},
	TypeSafety:    {},
	TypeOther:     {},
}

type AssignmentHistory struct {
	Responsible    string `json:"responsible" bson:"responsible"`
	AssignmentDate string `json:"assignmentDate" bson:"assignmentDate"`
}

type MaintenanceHistory struct {
	DateMaintenance string `json:"dateMaintenance" bson:"dateMaintenance"`
	NextMaintenance string `json:"nextMaintenance" bson:"nextMaintenance"`
}

type Tool struct {
	ID                string               `json:"id"`
	Code              string               `json:"code"`
	Name              string               `json:"name"`
	Type              string               `json:"type"`
	Status            string               `json:"status"`
	Responsible       string               `json:"responsible"`
	AssignmentDate    string               `json:"assignmentDate"`
	DateMaintenance   string               `json:"dateMaintenance"`
	NextMaintenance   string               `json:"nextMaintenance"`
	Location          string               `json:"location"`
	Notes             string               `json:"notes"`
	Deterioration     bool                 `json:"deterioration"`
	AssignmentHistory []AssignmentHistory  `json:"assignmentHistory"`
	MaintenanceRecord []MaintenanceHistory `json:"maintenanceRecord"`
}

type Filter struct {
	Search      string
	Status      string
	Responsible string
	Type        string
	Location    string
}

type ValidationErrors map[string]string

type ValidationError struct {
	Details ValidationErrors
}

func (e ValidationError) Error() string {
	return "validation failed"
}

func Normalize(input Tool) Tool {
	input.Code = strings.TrimSpace(input.Code)
	input.Name = strings.TrimSpace(input.Name)
	input.Type = strings.TrimSpace(input.Type)
	input.Status = strings.TrimSpace(input.Status)
	input.Responsible = strings.TrimSpace(input.Responsible)
	input.AssignmentDate = strings.TrimSpace(input.AssignmentDate)
	input.DateMaintenance = strings.TrimSpace(input.DateMaintenance)
	input.NextMaintenance = strings.TrimSpace(input.NextMaintenance)
	input.Location = strings.TrimSpace(input.Location)
	input.Notes = strings.TrimSpace(input.Notes)

	switch input.Status {
	case StatusActive:
		input.Responsible = ""
		input.AssignmentDate = ""
		input.DateMaintenance = ""
		input.NextMaintenance = ""
		input.Deterioration = false
	case StatusAssigned:
		input.DateMaintenance = ""
		input.NextMaintenance = ""
		input.Deterioration = false
	case StatusMaintenance:
		input.Responsible = ""
		input.AssignmentDate = ""
		input.Deterioration = false
	case StatusLost:
		input.DateMaintenance = ""
		input.NextMaintenance = ""
		input.Deterioration = false
	case StatusDamaged:
		input.DateMaintenance = ""
		input.NextMaintenance = ""
	}

	if input.AssignmentHistory == nil {
		input.AssignmentHistory = []AssignmentHistory{}
	}

	if input.MaintenanceRecord == nil {
		input.MaintenanceRecord = []MaintenanceHistory{}
	}

	return input
}

func Validate(input Tool) error {
	validationErrors := ValidationErrors{}

	if input.Code == "" {
		validationErrors["code"] = "El codigo es obligatorio"
	}
	if input.Name == "" {
		validationErrors["name"] = "El nombre es obligatorio"
	}
	if input.Location == "" {
		validationErrors["location"] = "La ubicacion es obligatoria"
	}
	if _, ok := validTypes[input.Type]; !ok {
		validationErrors["type"] = "El tipo no es valido"
	}
	if _, ok := validStatuses[input.Status]; !ok {
		validationErrors["status"] = "El estado no es valido"
	}

	switch input.Status {
	case StatusAssigned:
		if input.Responsible == "" {
			validationErrors["responsible"] = "El responsable es obligatorio para herramientas asignadas"
		}
		if input.AssignmentDate == "" {
			validationErrors["assignmentDate"] = "La fecha de asignacion es obligatoria para herramientas asignadas"
		}
	case StatusMaintenance:
		if input.DateMaintenance == "" {
			validationErrors["dateMaintenance"] = "La fecha de mantenimiento es obligatoria"
		}
		if input.NextMaintenance == "" {
			validationErrors["nextMaintenance"] = "La fecha del proximo mantenimiento es obligatoria"
		}
	}

	if len(validationErrors) > 0 {
		return ValidationError{Details: validationErrors}
	}

	return nil
}

func PrepareForCreate(input Tool) (Tool, error) {
	normalized := Normalize(input)
	if err := Validate(normalized); err != nil {
		return Tool{}, err
	}

	normalized = appendAssignmentHistory(Tool{}, normalized)
	normalized = appendMaintenanceHistory(Tool{}, normalized)

	return normalized, nil
}

func PrepareForUpdate(current Tool, incoming Tool) (Tool, error) {
	incoming.ID = current.ID
	normalized := Normalize(incoming)
	if err := Validate(normalized); err != nil {
		return Tool{}, err
	}

	normalized = appendAssignmentHistory(current, normalized)
	normalized = appendMaintenanceHistory(current, normalized)

	if normalized.Status != StatusAssigned {
		normalized.AssignmentHistory = current.AssignmentHistory
	}

	if normalized.Status != StatusMaintenance {
		normalized.MaintenanceRecord = current.MaintenanceRecord
	}

	return normalized, nil
}

func appendAssignmentHistory(current Tool, next Tool) Tool {
	if next.Status != StatusAssigned || next.Responsible == "" || next.AssignmentDate == "" {
		if next.AssignmentHistory == nil {
			next.AssignmentHistory = current.AssignmentHistory
		}
		return next
	}

	history := append([]AssignmentHistory{}, current.AssignmentHistory...)
	newEntry := AssignmentHistory{
		Responsible:    next.Responsible,
		AssignmentDate: next.AssignmentDate,
	}

	if shouldAppendAssignment(history, newEntry) {
		history = append(history, newEntry)
	}

	next.AssignmentHistory = capAssignmentHistory(history)
	return next
}

func appendMaintenanceHistory(current Tool, next Tool) Tool {
	if next.Status != StatusMaintenance || next.DateMaintenance == "" || next.NextMaintenance == "" {
		if next.MaintenanceRecord == nil {
			next.MaintenanceRecord = current.MaintenanceRecord
		}
		return next
	}

	records := append([]MaintenanceHistory{}, current.MaintenanceRecord...)
	newRecord := MaintenanceHistory{
		DateMaintenance: next.DateMaintenance,
		NextMaintenance: next.NextMaintenance,
	}

	if shouldAppendMaintenance(records, newRecord) {
		records = append(records, newRecord)
	}

	next.MaintenanceRecord = capMaintenanceHistory(records)
	return next
}

func shouldAppendAssignment(history []AssignmentHistory, next AssignmentHistory) bool {
	if len(history) == 0 {
		return true
	}

	last := history[len(history)-1]
	return !(last.Responsible == next.Responsible && last.AssignmentDate == next.AssignmentDate)
}

func shouldAppendMaintenance(history []MaintenanceHistory, next MaintenanceHistory) bool {
	if len(history) == 0 {
		return true
	}

	last := history[len(history)-1]
	return !(last.DateMaintenance == next.DateMaintenance && last.NextMaintenance == next.NextMaintenance)
}

func capAssignmentHistory(history []AssignmentHistory) []AssignmentHistory {
	if len(history) <= 10 {
		return history
	}

	return history[len(history)-10:]
}

func capMaintenanceHistory(history []MaintenanceHistory) []MaintenanceHistory {
	if len(history) <= 10 {
		return history
	}

	return history[len(history)-10:]
}

func ParseID(id string) error {
	if strings.TrimSpace(id) == "" {
		return errors.New("id is required")
	}

	return nil
}
