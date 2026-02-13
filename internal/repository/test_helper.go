package repository

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/namnv2496/mocktool/internal/configs"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// TestHelper provides utilities for repository integration tests
type TestHelper struct {
	Client *mongo.Client
	DB     *mongo.Database
	ctx    context.Context
}

// SetupTestDB creates a test database connection
// It uses MONGO_URI from environment or defaults to localhost
func SetupTestDB(t *testing.T) *TestHelper {
	t.Helper()

	// Get MongoDB URI from environment or use default
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://root:rootpassword@localhost:27017/mocktool?authSource=admin"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create MongoDB client
	clientOpts := options.Client().
		ApplyURI(mongoURI).
		SetMaxPoolSize(10).
		SetMinPoolSize(1).
		SetConnectTimeout(5 * time.Second).
		SetServerSelectionTimeout(5 * time.Second)

	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		t.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	// Ping to verify connection
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		t.Fatalf("Failed to ping MongoDB: %v", err)
	}

	// Use a test-specific database
	dbName := fmt.Sprintf("mocktool_test_%d", time.Now().UnixNano())
	db := client.Database(dbName)

	return &TestHelper{
		Client: client,
		DB:     db,
		ctx:    context.Background(),
	}
}

// Cleanup drops the test database and closes the connection
func (h *TestHelper) Cleanup(t *testing.T) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Drop the test database
	if err := h.DB.Drop(ctx); err != nil {
		t.Logf("Warning: failed to drop test database: %v", err)
	}

	// Close the client connection
	if err := h.Client.Disconnect(ctx); err != nil {
		t.Logf("Warning: failed to disconnect MongoDB client: %v", err)
	}
}

// GetContext returns the test context
func (h *TestHelper) GetContext() context.Context {
	return h.ctx
}

// LoadTestConfig creates a test configuration
func LoadTestConfig() *configs.Config {
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://root:rootpassword@localhost:27017/mocktool_test?authSource=admin"
	}

	return &configs.Config{
		MongoDB: configs.MongoDB{
			URI:      mongoURI,
			Database: "mocktool_test",
		},
	}
}
