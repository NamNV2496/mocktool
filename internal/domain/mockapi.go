package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type SequenceResponse struct {
	From    int      `bson:"from" json:"from"`
	To      int      `bson:"to" json:"to"`
	Output  bson.Raw `bson:"output,omitempty" json:"output"`
	Headers bson.Raw `bson:"headers,omitempty" json:"headers"`
	Latency int64    `bson:"latency" json:"latency"`
}

type MockAPI struct {
	ID           primitive.ObjectID `bson:"_id" json:"id"`
	FeatureName  string             `bson:"feature_name" json:"feature_name" validate:"required,no_spaces"`
	ScenarioName string             `bson:"scenario_name" json:"scenario_name" validate:"required,no_spaces"`
	Name         string             `bson:"name" json:"name" validate:"required,no_spaces"`
	Description  string             `bson:"description" json:"description"`
	IsActive     bool               `bson:"is_active" json:"is_active"`
	BaseURL      string             `bson:"base_url" json:"base_url"`
	Path         string             `bson:"path" json:"path" validate:"required,no_spaces"`
	Method       string             `bson:"method" json:"method" validate:"required,no_spaces"`
	Input        bson.Raw           `bson:"input,omitempty" json:"input"`
	HashInput    string             `bson:"hash_input" json:"hash_input"`
	Headers      bson.Raw           `bson:"headers,omitempty" json:"headers"`
	Output       bson.Raw           `bson:"output,omitempty" json:"output"`
	Latency      int64              `bson:"latency" json:"latency"`
	Responses    []SequenceResponse `bson:"responses,omitempty" json:"responses"`
	CreatedAt    time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt    time.Time          `bson:"updated_at" json:"updated_at"`
}
