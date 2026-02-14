package repository

import (
	"context"
	"time"

	"github.com/namnv2496/mocktool/internal/configs"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type IBaseRepository interface {
	Insert(ctx context.Context, doc interface{}) error
	FindByID(ctx context.Context, id int64, out interface{}) error
	FindMany(ctx context.Context, filter bson.M, out interface{}) error
	FindManyWithPagination(ctx context.Context, filter bson.M, skip int64, limit int64, out interface{}) error
	Count(ctx context.Context, filter bson.M) (int64, error)
	UpdateByID(ctx context.Context, id int64, update bson.M) error
	UpdateByObjectID(ctx context.Context, id primitive.ObjectID, update bson.M) error
	UpdateOne(ctx context.Context, filter bson.M, update bson.M) error
	UpdateMany(ctx context.Context, filter bson.M, update bson.M) error
	FindOne(ctx context.Context, filter bson.M, out interface{}) error
	DeleteOne(ctx context.Context, id primitive.ObjectID) (*mongo.DeleteResult, error)
	DeleteMany(ctx context.Context, filter bson.M) (*mongo.DeleteResult, error)
	DeleteOneByFilter(ctx context.Context, filter bson.M) error
}
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

func (_self *BaseRepository) Insert(ctx context.Context, doc interface{}) error {
	_, err := _self.col.InsertOne(ctx, doc)
	return err
}

func (_self *BaseRepository) FindByID(ctx context.Context, id int64, out interface{}) error {
	return _self.col.FindOne(ctx, bson.M{"_id": id}).Decode(out)
}

func (_self *BaseRepository) FindMany(
	ctx context.Context,
	filter bson.M,
	out interface{},
) error {
	cursor, err := _self.col.Find(ctx, filter)
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)

	return cursor.All(ctx, out)
}

func (_self *BaseRepository) FindManyWithPagination(
	ctx context.Context,
	filter bson.M,
	skip int64,
	limit int64,
	out interface{},
) error {
	opts := options.Find().
		SetSkip(skip).
		SetLimit(limit).
		SetSort(bson.D{{Key: "created_at", Value: -1}}) // newest first

	cursor, err := _self.col.Find(ctx, filter, opts)
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)

	return cursor.All(ctx, out)
}

func (_self *BaseRepository) Count(ctx context.Context, filter bson.M) (int64, error) {
	return _self.col.CountDocuments(ctx, filter)
}

func (_self *BaseRepository) UpdateByID(
	ctx context.Context,
	id int64,
	update bson.M,
) error {
	update["updated_at"] = now()

	_, err := _self.col.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{"$set": update},
	)
	return err
}

func (_self *BaseRepository) UpdateByObjectID(
	ctx context.Context,
	id primitive.ObjectID,
	update bson.M,
) error {
	update["updated_at"] = now()

	_, err := _self.col.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{"$set": update},
	)
	return err
}

func (_self *BaseRepository) UpdateOne(
	ctx context.Context,
	filter bson.M,
	update bson.M,
) error {
	_, err := _self.col.UpdateOne(ctx, filter, update)
	return err
}

func (_self *BaseRepository) UpdateMany(
	ctx context.Context,
	filter bson.M,
	update bson.M,
) error {
	_, err := _self.col.UpdateMany(ctx, filter, update)
	return err
}

func (_self *BaseRepository) FindOne(
	ctx context.Context,
	filter bson.M,
	out interface{},
) error {
	return _self.col.FindOne(ctx, filter).Decode(out)
}

func (_self *BaseRepository) DeleteOne(
	ctx context.Context,
	id primitive.ObjectID,
) (*mongo.DeleteResult, error) {
	return _self.col.DeleteOne(
		ctx,
		bson.M{"_id": id},
	)
}

func (_self *BaseRepository) DeleteOneByFilter(
	ctx context.Context,
	filter bson.M,
) error {
	_, err := _self.col.DeleteOne(
		ctx,
		filter,
	)
	return err
}
func (_self *BaseRepository) DeleteMany(ctx context.Context, filter bson.M) (*mongo.DeleteResult, error) {
	return _self.col.DeleteMany(ctx, filter)
}
