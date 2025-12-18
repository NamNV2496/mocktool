package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

type Scenario struct {
	ID          int64     `bson:"_id" json:"id"`
	Name        string    `bson:"name" json:"name"`
	Description string    `bson:"description" json:"description"`
	IsActive    bool      `bson:"is_active" json:"is_active"`
	CreatedAt   time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time `bson:"updated_at" json:"updated_at"`
}

func (_self Scenario) ToMap() bson.M {
	update := bson.M{}
	if _self.Name != "" {
		update["name"] = _self.Name
	}
	if _self.Description != "" {
		update["description"] = _self.Description
	}

	update["is_active"] = _self.IsActive
	update["updated_at"] = time.Now().UTC()

	return update
}
