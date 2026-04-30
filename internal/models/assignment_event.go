package models

import "go.mongodb.org/mongo-driver/v2/bson"

type AssignmentAction string

const (
	AssignmentActionAssigned AssignmentAction = "assigned"
	AssignmentActionReturned AssignmentAction = "returned"
)

type AssignmentEvent struct {
	ID             bson.ObjectID    `bson:"_id,omitempty" json:"id"`
	ToolID         string           `bson:"toolId" json:"toolId"`
	ToolCode       string           `bson:"toolCode" json:"toolCode"`
	ToolName       string           `bson:"toolName" json:"toolName"`
	WorkerID       string           `bson:"workerId" json:"workerId"`
	WorkerName     string           `bson:"workerName" json:"workerName"`
	Action         AssignmentAction `bson:"action" json:"action"`
	AssignmentDate string           `bson:"assignmentDate" json:"assignmentDate"`
	CreatedAt      string           `bson:"createdAt" json:"createdAt"`
}
