package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// AccountScenario maps which scenario is active for a specific account (or globally)
type AccountScenario struct {
	ID          primitive.ObjectID `bson:"_id" json:"id"`
	FeatureName string             `bson:"feature_name" json:"feature_name"`
	ScenarioID  primitive.ObjectID `bson:"scenario_id" json:"scenario_id"`
	AccountId   *string            `bson:"account_id,omitempty" json:"account_id,omitempty"` // null/empty for global
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
}

func (_self AccountScenario) ToMap() bson.M {
	update := bson.M{}
	if _self.FeatureName != "" {
		update["feature_name"] = _self.FeatureName
	}
	if !_self.ScenarioID.IsZero() {
		update["scenario_id"] = _self.ScenarioID
	}
	if _self.AccountId != nil {
		update["account_id"] = *_self.AccountId
	}
	update["updated_at"] = time.Now().UTC()

	return update
}
