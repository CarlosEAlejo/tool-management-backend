package models

import "go.mongodb.org/mongo-driver/v2/bson"

type Trabajador struct {
	ID        bson.ObjectID `bson:"_id,omitempty" json:"id"`
	FirstName string        `bson:"firstName" json:"firstName"`
	LastName  string        `bson:"lastName" json:"lastName"`
	Position  string        `bson:"position" json:"position"`
	Email     string        `bson:"email" json:"email"`
	Phone     string        `bson:"phone" json:"phone"`
	Notes     string        `bson:"notes" json:"notes"`
}
