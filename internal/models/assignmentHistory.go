package models

type AssignmentHistory struct {
	Responsible    string `bson:"responsible" json:"responsible"`
	AssignmentDate string `bson:"assignmentDate" json:"assignmentDate"`
}
