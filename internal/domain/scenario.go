package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Scenario struct {
	ID          primitive.ObjectID `bson:"_id" json:"id"`
	FeatureName string             `bson:"feature_name" json:"feature_name"`
	Name        string             `bson:"name" json:"name"`
	Description string             `bson:"description" json:"description"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
}

func (_self Scenario) ToMap() bson.M {
	update := bson.M{}
	if _self.Name != "" {
		update["name"] = _self.Name
	}
	if _self.Description != "" {
		update["description"] = _self.Description
	}
	update["updated_at"] = time.Now().UTC()

	return update
}
