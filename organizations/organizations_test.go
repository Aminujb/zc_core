package organizations

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"zuri.chat/zccore/utils"
)

var configs = utils.NewConfigurations()
var orgs = NewOrganizationHandler(configs, nil)
const defaultUser string = "testUser@gmail.com"

func TestMain(m *testing.M) {
	// load .env file if it exists
	err := godotenv.Load("../.testenv")
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	fmt.Println("Environment variables successfully loaded. Starting application...")

	if err = utils.ConnectToDB(os.Getenv("CLUSTER_URL")); err != nil {
		log.Fatal("Could not connect to MongoDB")
	}
	fmt.Printf("\n\n")

	err = setUpUserAccount()
	if err != nil {
		log.Fatal("User account exists")
	}

	exitVal := m.Run()

	// drop database after running all tests
	ctx := context.TODO()
	utils.GetDefaultMongoClient().Database(os.Getenv("DB_NAME")).Drop(ctx)

    os.Exit(exitVal)
}

func TestCreateOrganization(t *testing.T) {
	t.Run("test for invalid json request body", func(t *testing.T) {
		req, err := http.NewRequest("POST", "/organizations", nil)
		if err != nil {
			t.Fatal(err)
		}
		response := httptest.NewRecorder()
		orgs.Create(response, req)
		assertStatusCode(t, response.Code, http.StatusBadRequest)
	})

	t.Run("test for wrong request body key", func(t *testing.T) {
		var requestBody = []byte(`{"creat_email": "badmailformat.xyz"}`)

		req, err := http.NewRequest("POST", "/organizations", bytes.NewBuffer(requestBody))
		if err != nil {
			t.Fatal(err)
		}
		response := httptest.NewRecorder()
		orgs.Create(response, req)
		assertStatusCode(t, response.Code, http.StatusBadRequest)
		assertResponseMessage(t, parseResponse(response)["message"].(string), "invalid email format : ")
	})

	t.Run("test for bad email format", func(t *testing.T) {
		var requestBody = []byte(`{"creator_email": "badmailformat.xyz"}`)

		req, err := http.NewRequest("POST", "/organizations", bytes.NewBuffer(requestBody))
		if err != nil {
			t.Fatal(err)
		}
		response := httptest.NewRecorder()
		orgs.Create(response, req)
		assertStatusCode(t, response.Code, http.StatusBadRequest)
		assertResponseMessage(t, parseResponse(response)["message"].(string), "invalid email format : badmailformat.xyz")
	})

	t.Run("test for non existent user", func(t *testing.T) {
		var requestBody = []byte(`{"creator_email": "notuser@gmail.com"}`)

		req, err := http.NewRequest("POST", "/organizations", bytes.NewBuffer(requestBody))
		if err != nil {
			t.Fatal(err)
		}
		response := httptest.NewRecorder()
		orgs.Create(response, req)
		assertStatusCode(t, response.Code, http.StatusBadRequest)
		assertResponseMessage(t, parseResponse(response)["message"].(string), "user with this email does not exist")
	})

	/*
		The below test requires that the user has to exist first!
	*/

	t.Run("test for successful organization creation", func(t *testing.T) {
		var requestBody = []byte(`{"creator_email": "testUser@gmail.com"}`)

		req, err := http.NewRequest("POST", "/organizations", bytes.NewBuffer(requestBody))
		if err != nil {
			t.Fatal(err)
		}
		response := httptest.NewRecorder()
		orgs.Create(response, req)
		assertStatusCode(t, response.Code, http.StatusOK)
	})
}

func TestGetOrganization(t *testing.T) {
	t.Run("test for invalid id fails", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/organizations/12345", nil)
		if err != nil {
			t.Fatal(err)
		}
		response := httptest.NewRecorder()
		orgs.GetOrganization(response, req)
		assertStatusCode(t, response.Code, http.StatusBadRequest)
	})
}

func assertStatusCode(t *testing.T, got, expected int) {
	if got != expected {
		t.Errorf("got status %d expected status %d", got, expected)
	}
}

func assertResponseMessage(t *testing.T, got, expected string) {
	if got != expected {
		t.Errorf("got message: %q expected: %q", got, expected)
	}
}

func parseResponse(w *httptest.ResponseRecorder) map[string]interface{} {
	res := make(map[string]interface{})
	json.NewDecoder(w.Body).Decode(&res)
	return res
}

func setUpUserAccount() error{
	type User struct {
		ID            string 	`json:"_id,omitempty" bson:"_id,omitempty"` 
		Email         string	`json:"email" bson:"email"` 
		IsVerified    bool  	`json:"is_verified" bson:"is_verified"`   
		Deactivated   bool  	`json:"deactivated" bson:"deactivated"`    
	}

	user := User{
		Email: defaultUser,
		Deactivated: false,
		IsVerified: true,
	}

	result, _ := utils.GetMongoDBDoc(UserCollectionName, bson.M{"email": user.Email})
	if result != nil {
		return fmt.Errorf("user %s exists", user.Email)
	}

	detail, _ := utils.StructToMap(user)
	_, err := utils.CreateMongoDBDoc(UserCollectionName, detail)

	if err != nil {
		return err
	}
	
	return err
}
