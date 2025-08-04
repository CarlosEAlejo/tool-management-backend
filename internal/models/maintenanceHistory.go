package models

type MaintenanceHistory struct {
	DateMaintenance string `bson:"dateMaintenance" json:"dateMaintenance"`
	NextMaintenance string `bson:"nextMaintenance" json:"nextMaintenance"`
}
