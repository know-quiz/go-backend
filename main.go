package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/option"
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

func getFireStoreQuestions(w http.ResponseWriter, r *http.Request) {
	var questions []Question
	ctx := context.Background()

	// Firestore setup
	// Replace "path/to/your/service-account-file.json" with your service account file path
	// and "your-project-id" with your GCP project ID.

	d, _ := base64.StdEncoding.DecodeString(os.Getenv("GCP_CREDS_JSON_BASE64"))
	//sa := option.WithCredentialsFile("/Users/dileep/Downloads/firestore-quiz-app-415922-d27bce37d853.json")
	client, err := firestore.NewClient(ctx, "quiz-app-415922", option.WithCredentialsJSON(d))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer client.Close()

	// Specify the collection
	collection := "quiz-questions"

	// Query all documents in the collection

	iter := client.Collection(collection).Documents(ctx)
	for {
		var question Question
		doc, err := iter.Next()
		if err != nil {
			break
		}
		if err := doc.DataTo(&question); err != nil {
			log.Printf("Failed to marshal document to struct: %v", err)
			continue
		}
		questions = append(questions, question)
	}

	// Set Content-Type header
	w.Header().Set("Content-Type", "application/json")
	// Encode and return the data as JSON
	if err := json.NewEncoder(w).Encode(questions); err != nil {
		http.Error(w, "Error encoding response to JSON", http.StatusInternalServerError)
	}
}

func main() {
	// Create a new HTTP server
	http.HandleFunc("/api/questions", getFireStoreQuestions)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
