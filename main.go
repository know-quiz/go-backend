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
	ID       string `json:"id"`
	Question string `json:"question"`
	Option1  string `json:"option1"`
	Option2  string `json:"option2"`
	Option3  string `json:"option3"`
	Option4  string `json:"option4"`
	Answer   string `json:"answer"`
}

type QuestionAnswer struct {
	Question          string `json:"question"`
	AnsweredCorrectly bool   `json:"answeredCorrectly"`
}

type UserAnswer struct {
	UserID            string           `json:"userId"`
	AnsweredQuestions []QuestionAnswer `json:"answeredQuestions"`
}

type ApiResponse struct {
	Message string `json:"message"`
}

func createFirestoreClient(ctx context.Context) (*firestore.Client, error) {
	d, _ := base64.StdEncoding.DecodeString(os.Getenv("GCP_CREDS_JSON_BASE64"))
	return firestore.NewClient(ctx, "quiz-app-415922", option.WithCredentialsJSON(d))
}

func getFireStoreQuestions(w http.ResponseWriter, r *http.Request) {
	var questions []Question
	ctx := context.Background()
	client, err := createFirestoreClient(ctx)
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
		question.ID = string(doc.Ref.ID)
		questions = append(questions, question)
	}

	// Set Content-Type header
	w.Header().Set("Content-Type", "application/json")
	// Encode and return the data as JSON
	if err := json.NewEncoder(w).Encode(questions); err != nil {
		http.Error(w, "Error encoding response to JSON", http.StatusInternalServerError)
	}
}

// Writes a JSON response
func writeJsonResponse(w http.ResponseWriter, statusCode int, response ApiResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

func postUserAnswers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJsonResponse(w, http.StatusMethodNotAllowed, ApiResponse{"Unsupported method"})
		return
	}

	var userAnswer UserAnswer
	// Decode the request body
	err := json.NewDecoder(r.Body).Decode(&userAnswer)
	if err != nil {
		writeJsonResponse(w, http.StatusBadRequest, ApiResponse{err.Error()})
		return
	}

	// Basic validation
	if userAnswer.UserID == "" || len(userAnswer.AnsweredQuestions) == 0 {
		writeJsonResponse(w, http.StatusBadRequest, ApiResponse{"Missing required fields"})
		return
	}

	ctx := context.Background()
	client, err := createFirestoreClient(ctx)
	if err != nil {
		writeJsonResponse(w, http.StatusInternalServerError, ApiResponse{"Error writing to Firestore: " + err.Error()})
		return
	}
	defer client.Close()

	// Start a batch
	bulkWriter := client.BulkWriter(ctx)

	// Reference to the collection
	colRef := client.Collection("userAnswers")

	// Loop through answered questions and prepare writes
	for _, qa := range userAnswer.AnsweredQuestions {
		docRef := colRef.Doc(userAnswer.UserID).Collection("answeredQuestions").NewDoc()
		bulkWriter.Set(docRef, qa)
	}
	bulkWriter.Flush()
	writeJsonResponse(w, http.StatusOK, ApiResponse{"Answers recorded successfully"})
}

func getUserAnswers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJsonResponse(w, http.StatusMethodNotAllowed, ApiResponse{"Unsupported method"})
		return
	}

	// Extract the userId from query parameters
	userId := r.URL.Query().Get("userId")
	if userId == "" {
		writeJsonResponse(w, http.StatusBadRequest, ApiResponse{"Missing userId"})
		return
	}

	ctx := context.Background()
	client, err := createFirestoreClient(ctx)
	if err != nil {
		writeJsonResponse(w, http.StatusInternalServerError, ApiResponse{"Error writing to Firestore: " + err.Error()})
		return
	}
	defer client.Close()

	// Firestore query
	var userAnswers []QuestionAnswer
	iter := client.Collection("userAnswers").Doc(userId).Collection("answeredQuestions").Documents(ctx)
	for {
		doc, err := iter.Next()
		if err != nil {
			break
		}
		var qa QuestionAnswer
		if err := doc.DataTo(&qa); err != nil {
			writeJsonResponse(w, http.StatusInternalServerError, ApiResponse{"Error reading from Firestore: " + err.Error()})
			return
		}
		userAnswers = append(userAnswers, qa)
	}

	// Check for empty results
	if len(userAnswers) == 0 {
		writeJsonResponse(w, http.StatusNotFound, ApiResponse{"No answers found for user"})
		return
	}

	// Success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(userAnswers)
}

func main() {
	// Create a new HTTP server
	http.HandleFunc("GET /api/questions", getFireStoreQuestions)
	http.HandleFunc("POST /api/user/answers", postUserAnswers)
	http.HandleFunc("GET /api/user/answers", getUserAnswers)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
