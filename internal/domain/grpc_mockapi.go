package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type GRPCMockAPI struct {
	ID           primitive.ObjectID `bson:"_id" json:"id"`
	FeatureName  string             `bson:"feature_name" json:"feature_name" validate:"required,no_spaces"`
	ScenarioName string             `bson:"scenario_name" json:"scenario_name" validate:"required,no_spaces"`
	ServiceName  string             `bson:"service_name" json:"service_name" validate:"required"`
	MethodName   string             `bson:"method_name" json:"method_name" validate:"required"`
	HashInput    string             `bson:"hash_input" json:"hash_input"`
	Input        bson.Raw           `bson:"input,omitempty" json:"input"`
	Output       bson.Raw           `bson:"output,omitempty" json:"output"`
	StatusCode   int32              `bson:"status_code,omitempty" json:"status_code,omitempty"`
	Latency      int64              `bson:"latency" json:"latency"`
	IsActive     bool               `bson:"is_active" json:"is_active"`
	CreatedAt    time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt    time.Time          `bson:"updated_at" json:"updated_at"`
}
