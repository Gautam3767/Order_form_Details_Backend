package database

import (
	"context"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var mongoClient *mongo.Client
var mongoDB *mongo.Database
var brandCollection *mongo.Collection

// Connect initializes the MongoDB connection
func Connect() {
	mongoURI := os.Getenv("MONGODB_URI")
	dbName := os.Getenv("MONGODB_DATABASE")
	collectionName := os.Getenv("MONGODB_COLLECTION")

	if mongoURI == "" || dbName == "" || collectionName == "" {
		log.Fatal("MONGODB_URI, MONGODB_DATABASE, and MONGODB_COLLECTION must be set in the environment variables or .env file")
	}

	// Use context with timeout for connection attempt
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel() // Release resources associated with context

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatalf("Failed to create MongoDB client: %v", err)
	}

	// Ping the primary server to verify the connection.
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		log.Fatalf("Failed to connect to MongoDB (ping failed): %v", err)
	}

	log.Println("Successfully connected and pinged MongoDB.")

	mongoClient = client
	mongoDB = client.Database(dbName)
	brandCollection = mongoDB.Collection(collectionName)

	// --- Optional: Create Indexes ---
	// Create a unique index on the 'name' field in the background
	// It's good practice to ensure brand names are unique at the DB level
	go func() {
		indexModel := mongo.IndexModel{
			Keys:    map[string]interface{}{"name": 1}, // 1 for ascending order
			Options: options.Index().SetUnique(true).SetBackground(true),
		}
		_, err := brandCollection.Indexes().CreateOne(context.Background(), indexModel)
		if err != nil {
			// Log the error but don't necessarily crash the app
			// It might fail if the index already exists or if there are duplicate names before the index is created
			log.Printf("Warning: Could not create unique index on 'name': %v", err)
		} else {
			log.Println("Unique index on 'name' field ensured.")
		}
	}()

}

// GetDB returns the MongoDB database instance
// Deprecated: Prefer GetCollection for specific operations
func GetDB() *mongo.Database {
	return mongoDB
}

// GetCollection returns the specific MongoDB collection for brands
func GetCollection(name string) *mongo.Collection {
	// In this simple case, we only have one collection pre-defined
	if name == os.Getenv("MONGODB_COLLECTION") {
		return brandCollection
	}
	// If you had multiple collections, you could fetch them dynamically:
	// return mongoDB.Collection(name)
	log.Printf("Warning: Requested unknown collection '%s', returning default brand collection", name)
	return brandCollection // Or return nil/error
}

// Disconnect closes the MongoDB connection
// Call this on graceful shutdown if needed
func Disconnect() {
	if mongoClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := mongoClient.Disconnect(ctx); err != nil {
			log.Fatalf("Error disconnecting MongoDB: %v", err)
		}
		log.Println("MongoDB connection closed.")
	}
}
