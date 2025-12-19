package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MockAPI struct {
	ID           primitive.ObjectID `bson:"_id" json:"id"`
	FeatureName  string             `bson:"feature_name" json:"feature_name"`
	ScenarioName string             `bson:"scenario_name" json:"scenario_name"`
	Name         string             `bson:"name" json:"name"`
	Description  string             `bson:"description" json:"description"`
	IsActive     bool               `bson:"is_active" json:"is_active"`
	Path         string             `bson:"path" json:"path"`             // path
	RegexPath    string             `bson:"regex_path" json:"regex_path"` // regex_path
	HashInput    bson.Raw           `bson:"hash_input" json:"hash_input"` // original JSON input for hashing
	HashValue    string             `bson:"hash_value" json:"-"`          // computed hash value (not exposed in JSON)
	Output       bson.Raw           `bson:"output" json:"output"`         // json response
	CreatedAt    time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt    time.Time          `bson:"updated_at" json:"updated_at"`
}

func (_self MockAPI) ToMap() bson.M {
	update := bson.M{}

	if _self.Name != "" {
		update["name"] = _self.Name
	}

	if _self.Description != "" {
		update["description"] = _self.Description
	}

	if _self.Path != "" {
		update["path"] = _self.Path
	}
	if _self.RegexPath != "" {
		update["regex_path"] = _self.RegexPath
	}
	if _self.HashInput != nil && len(_self.HashInput) > 0 {
		update["hash_input"] = _self.HashInput
	}
	if _self.HashValue != "" {
		update["hash_value"] = _self.HashValue
	}

	if _self.Output != nil && len(_self.Output) > 0 {
		update["output"] = _self.Output
	}

	// bool is tricky: always include
	update["is_active"] = _self.IsActive

	update["updated_at"] = time.Now().UTC()

	return update
}
