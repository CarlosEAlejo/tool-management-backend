package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Herramienta struct {
	ID                primitive.ObjectID   `bson:"_id,omitempty" json:"id"`
	Code              string               `bson:"code" json:"code"`
	Name              string               `bson:"name" json:"name"`
	Type              string               `bson:"type" json:"type"`
	Status            string               `bson:"status" json:"status"`
	Responsible       string               `bson:"responsible" json:"responsible"`
	AssignmentDate    string               `bson:"assignmentDate" json:"assignmentDate"`
	DateMaintenance   string               `bson:"dateMaintenance" json:"dateMaintenance"`
	NextMaintenance   string               `bson:"nextMaintenance" json:"nextMaintenance"`
	Location          string               `bson:"location" json:"location"`
	Notes             string               `bson:"notes" json:"notes"`
	Deterioration     bool                 `bson:"deterioration" json:"deterioration"`
	AssignmentHistory []AssignmentHistory  `bson:"assignmentHistory" json:"assignmentHistory"`
	MaintenanceRecord []MaintenanceHistory `bson:"maintenanceRecord" json:"maintenanceRecord"`
}
