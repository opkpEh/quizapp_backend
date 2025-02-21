package main

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"log"
	"net/http"
	"os"
)

var (
	uri    string
	client *mongo.Client
	ctx    context.Context
)

func mongoInit() {
	if uri = os.Getenv("MONGODB_URI"); uri == "" {
		log.Fatal("You must set your 'MONGODB_URI' environment variable. See\n\t https://docs.mongodb.com/drivers/go/current/usage-examples/")
	}
	ctx = context.Background()

	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	opts := options.Client().ApplyURI(uri).SetServerAPIOptions(serverAPI)

	var err error
	client, err = mongo.Connect(opts)
	if err != nil {
		panic(err)
	}

	var result bson.M
	if err := client.Database("admin").RunCommand(ctx, bson.D{{"ping", 1}}).Decode(&result); err != nil {
		panic(err)
	}
	fmt.Println("Pinged your deployment. You successfully connected to MongoDB!")
}

type question struct {
	Question string   `json:"question" bson:"question"`
	Options  []string `json:"options" bson:"options"`
	Answer   string   `json:"answer" bson:"answer"`
	Category string   `json:"category" bson:"category"`
}

func getQuestions(c *gin.Context) {
	collection := client.Database("quizapp").Collection("questions")

	cursor, err := collection.Find(ctx, bson.D{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch questions from database"})
		return
	}
	defer func(cursor *mongo.Cursor, ctx context.Context) {
		err := cursor.Close(ctx)
		if err != nil {

		}
	}(cursor, ctx)

	var results []question

	if err = cursor.All(ctx, &results); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode questions from database"})
		return
	}
	c.IndentedJSON(http.StatusOK, results)
}

func addQuestion(c *gin.Context) {
	var newQuestion question

	if err := c.BindJSON(&newQuestion); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if newQuestion.Category == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Category is required"})
		return
	}

	collection := client.Database("quizapp").Collection("questions")
	_, err := collection.InsertOne(ctx, newQuestion)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store question in database"})
		return
	}
	c.IndentedJSON(http.StatusCreated, newQuestion)
}

// Get questions by category
func getQuestionsByCategory(c *gin.Context) {
	category := c.Param("category")

	collection := client.Database("quizapp").Collection("questions")

	filter := bson.D{{"category", category}}
	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch questions from database"})
		return
	}
	defer func(cursor *mongo.Cursor, ctx context.Context) {
		err := cursor.Close(ctx)
		if err != nil {

		}
	}(cursor, ctx)

	var results []question
	if err = cursor.All(ctx, &results); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode questions from database"})
		return
	}

	c.IndentedJSON(http.StatusOK, results)
}

func main() {
	mongoInit()
	router := gin.Default()

	router.GET("/questions", getQuestions)
	router.GET("/questions/category/:category", getQuestionsByCategory)
	router.POST("/questions", addQuestion)

	err := router.Run("localhost:8080")
	if err != nil {
		return
	}

	defer func() {
		if err = client.Disconnect(ctx); err != nil {
			panic(err)
		}
	}()
}
