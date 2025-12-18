package repository

import (
	"context"
	"time"

	"github.com/namnv2496/mocktool/internal/configs"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type BaseRepository struct {
	col *mongo.Collection
}

func NewMongoConnect(conf *configs.Config) (*mongo.Client, *mongo.Database, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOpts := options.Client().
		ApplyURI(conf.MongoDB.URI).
		SetMaxPoolSize(50).
		SetMinPoolSize(5).
		SetConnectTimeout(5 * time.Second).
		SetServerSelectionTimeout(5 * time.Second)

	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, nil, err
	}

	// Ping primary
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		return nil, nil, err
	}

	db := client.Database(conf.MongoDB.Database)
	return client, db, nil
}

func NewBaseRepository(col *mongo.Collection) *BaseRepository {
	return &BaseRepository{col: col}
}

func now() time.Time {
	return time.Now().UTC()
}

/* ---------- basic ops ---------- */

func (r *BaseRepository) Insert(ctx context.Context, doc interface{}) error {
	_, err := r.col.InsertOne(ctx, doc)
	return err
}

func (r *BaseRepository) FindByID(ctx context.Context, id int64, out interface{}) error {
	return r.col.FindOne(ctx, bson.M{"_id": id}).Decode(out)
}

func (r *BaseRepository) FindMany(
	ctx context.Context,
	filter bson.M,
	out interface{},
) error {
	cursor, err := r.col.Find(ctx, filter)
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)

	return cursor.All(ctx, out)
}

func (r *BaseRepository) UpdateByID(
	ctx context.Context,
	id int64,
	update bson.M,
) error {
	update["updated_at"] = now()

	_, err := r.col.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{"$set": update},
	)
	return err
}
