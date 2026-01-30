package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// LoadTestStep represents a single API call in the load test scenario
type LoadTestStep struct {
	Name          string            `bson:"name" json:"name"`
	Method        string            `bson:"method" json:"method"`
	Path          string            `bson:"path" json:"path"`
	Headers       map[string]string `bson:"headers,omitempty" json:"headers,omitempty"`
	Body          string            `bson:"body,omitempty" json:"body,omitempty"`
	SaveVariables map[string]string `bson:"save_variables,omitempty" json:"save_variables,omitempty"` // Map of variable_name -> json_path
	ExpectStatus  int               `bson:"expect_status,omitempty" json:"expect_status,omitempty"`
}

// LoadTestAccount represents a user account for load testing
type LoadTestAccount struct {
	Username string            `bson:"username" json:"username"`
	Password string            `bson:"password" json:"password"`
	Extra    map[string]string `bson:"extra,omitempty" json:"extra,omitempty"` // Additional fields
}

// LoadTestScenario represents a load test scenario stored in DB
type LoadTestScenario struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Name        string             `bson:"name" json:"name"`
	Description string             `bson:"description,omitempty" json:"description,omitempty"`
	Concurrency int                `bson:"concurrency" json:"concurrency"`
	Accounts    string             `bson:"accounts" json:"accounts"` // Comma-separated username-password pairs
	Steps       []LoadTestStep     `bson:"steps" json:"steps"`
	IsActive    bool               `bson:"is_active" json:"is_active"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
}
