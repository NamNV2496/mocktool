package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MockAPI struct {
	ID           primitive.ObjectID `bson:"_id" json:"id"`
	FeatureName  string             `bson:"feature_name" json:"feature_name" validate:"required,no_spaces"`
	ScenarioName string             `bson:"scenario_name" json:"scenario_name" validate:"required,no_spaces"`
	Name         string             `bson:"name" json:"name" validate:"required,no_spaces"`
	Description  string             `bson:"description" json:"description"`
	IsActive     bool               `bson:"is_active" json:"is_active"`
	Path         string             `bson:"path" json:"path" validate:"required,no_spaces"`
	Method       string             `bson:"method" json:"method" validate:"required,no_spaces"`
	Input        bson.Raw           `bson:"input" json:"input"`
	HashInput    string             `bson:"hash_input" json:"hash_input"`
	Headers      bson.Raw           `bson:"headers" json:"headers"`
	Output       bson.Raw           `bson:"output" json:"output"`
	CreatedAt    time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt    time.Time          `bson:"updated_at" json:"updated_at"`
}
