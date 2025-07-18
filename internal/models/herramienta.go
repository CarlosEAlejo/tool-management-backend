package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Herramienta struct {
	ID                 primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Code               string             `bson:"code" json:"code"`
	Name               string             `bson:"name" json:"name"`
	Type               string             `bson:"type" json:"type"`
	Status             string             `bson:"status" json:"status"`
	Responsible        string             `bson:"responsible" json:"responsible"`
	AssignmentDate     string             `bson:"assignmentDate" json:"assignmentDate"`
	NextMaintenance    string             `bson:"nextMaintenance" json:"nextMaintenance"`
	Location           string             `bson:"location" json:"location"`
	Notes              string             `bson:"notes" json:"notes"`
	MaintenanceHistory []string           `bson:"maintenanceHistory" json:"maintenanceHistory"`
}
