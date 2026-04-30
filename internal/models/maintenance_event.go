package models

import "go.mongodb.org/mongo-driver/v2/bson"

type MaintenanceAction string

const (
	MaintenanceActionScheduled MaintenanceAction = "scheduled"
	MaintenanceActionCompleted MaintenanceAction = "completed"
)

type MaintenanceEvent struct {
	ID              bson.ObjectID     `bson:"_id,omitempty" json:"id"`
	ToolID          string            `bson:"toolId" json:"toolId"`
	ToolCode        string            `bson:"toolCode" json:"toolCode"`
	ToolName        string            `bson:"toolName" json:"toolName"`
	Action          MaintenanceAction `bson:"action" json:"action"`
	DateMaintenance string            `bson:"dateMaintenance" json:"dateMaintenance"`
	NextMaintenance string            `bson:"nextMaintenance" json:"nextMaintenance"`
	CreatedAt       string            `bson:"createdAt" json:"createdAt"`
}
