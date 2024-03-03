package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Question struct {
	ID       string `json:"id" bson:"_id,omitempty"`
	Question string `json:"question"`
	Option1  string `json:"option1"`
	Option2  string `json:"option2"`
	Option3  string `json:"option3"`
	Option4  string `json:"option4"`
	Answer   string `json:"answer"`
}

var client *mongo.Client

func getQuestions(w http.ResponseWriter, r *http.Request) {
	var questions []Question
	collection := client.Database("quiz_app").Collection("questions")
	ctx := context.TODO()
	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer cursor.Close(ctx)
	for cursor.Next(ctx) {
		var question Question
		cursor.Decode(&question)
		questions = append(questions, question)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(questions)
}

func main() {
	var err error
	client, err = mongo.Connect(context.TODO(), options.Client().ApplyURI("mongodb+srv://app_user:h1yTCK2XApBjAhDu@main.zyalogv.mongodb.net/?retryWrites=true&w=majority&appName=Main"))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(context.TODO())

	http.HandleFunc("/api/questions", getQuestions)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
