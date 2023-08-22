package main

import (
	"context"
	"log"
	"os"

	"github.com/gergpol1998/gin-mongo-api/routes"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	mongoURI := os.Getenv("MONGO_URI")
	clientOptions := options.Client().ApplyURI(mongoURI)
	client, err := mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		log.Fatal(err)
	}

	db := client.Database("user_db")
	userCollection := db.Collection("users")

	r := routes.SetupRouter(userCollection)

	port := os.Getenv("PORT")
	if port == "" {
		port = "6000"
	}

	r.Run(":" + port)
}
