package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// LoadTestStep represents a single API call in the load test scenario
type LoadTestStep struct {
	Name             string            `bson:"name" json:"name"`
	Method           string            `bson:"method" json:"method"`
	Path             string            `bson:"path" json:"path"`
	Headers          map[string]string `bson:"headers,omitempty" json:"headers,omitempty"`
	Body             string            `bson:"body,omitempty" json:"body,omitempty"`
	SaveVariables    map[string]string `bson:"save_variables,omitempty" json:"save_variables,omitempty"` // Map of variable_name -> json_path
	ExpectStatus     int               `bson:"expect_status,omitempty" json:"expect_status,omitempty"`
	WaitAfterSeconds int               `bson:"wait_after_seconds,omitempty" json:"wait_after_seconds,omitempty"` // Wait duration in seconds after step execution
	RetryForSeconds  int               `bson:"retry_for_seconds,omitempty" json:"retry_for_seconds,omitempty"`   // Retry duration in seconds if request fails
	MaxRetryTimes    int               `bson:"max_retry_times,omitempty" json:"max_retry_times,omitempty"`       // Maximum number of retry attempts (0 = unlimited within time limit)
	Condition        string            `bson:"condition,omitempty" json:"condition,omitempty"`                   // Condition to execute this step (e.g., "{{need_payment}} == true")
	UploadFiles      []UploadFile      `bson:"upload_files,omitempty" json:"upload_files,omitempty"`             // Files to upload in this request
}

// UploadFile represents a file to be uploaded in a request
type UploadFile struct {
	FieldName string `bson:"field_name" json:"field_name"` // Form field name (e.g., "file", "image")
	FilePath  string `bson:"file_path" json:"file_path"`   // Path to the file to upload (supports {{variables}})
	FileName  string `bson:"file_name,omitempty" json:"file_name,omitempty"` // Optional custom filename
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
	Accounts    string             `bson:"accounts" json:"accounts"` // Comma-separated username-password pairs
	Steps       []LoadTestStep     `bson:"steps" json:"steps"`
	IsActive    bool               `bson:"is_active" json:"is_active"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
}
