package models

import "go.mongodb.org/mongo-driver/v2/bson"

type Herramienta struct {
	ID                bson.ObjectID        `bson:"_id,omitempty" json:"id"`
	Code              string               `bson:"code" json:"code"`
	Name              string               `bson:"name" json:"name"`
	Type              string               `bson:"type" json:"type"`
	Status            string               `bson:"status" json:"status"`
	ResponsibleID     string               `bson:"responsibleId" json:"responsibleId"`
	Responsible       string               `bson:"responsible" json:"responsible"`
	AssignmentDate    string               `bson:"assignmentDate" json:"assignmentDate"`
	DateMaintenance   string               `bson:"dateMaintenance" json:"dateMaintenance"`
	NextMaintenance   string               `bson:"nextMaintenance" json:"nextMaintenance"`
	PurchaseDate      string               `bson:"purchaseDate" json:"purchaseDate"`
	Price             float64              `bson:"price" json:"price"`
	Location          string               `bson:"location" json:"location"`
	Notes             string               `bson:"notes" json:"notes"`
	Deterioration     bool                 `bson:"deterioration" json:"deterioration"`
	AssignmentHistory []AssignmentHistory  `bson:"assignmentHistory" json:"assignmentHistory"`
	MaintenanceRecord []MaintenanceHistory `bson:"maintenanceRecord" json:"maintenanceRecord"`
}
