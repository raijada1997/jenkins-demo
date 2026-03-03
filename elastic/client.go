package elastic

import (
	"crypto/tls"
	"log"
	"net/http"

	"os"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/joho/godotenv"
)

var ES *elasticsearch.Client // using this variable which stores pointer to elasticsearch.Client, we can connect to server and perform actions

const IndexName = "stages-v12"

// to do secret to go in env
func InitElastic() {

	godotenv.Load()

	esURL := os.Getenv("ELASTIC_URL")
	username := os.Getenv("USERNAME")
	password := os.Getenv("PASSWORD")

	cfg := elasticsearch.Config{
		Addresses: []string{
			esURL,
		},
		Username: username,
		Password: password,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		log.Fatalf("Error creating ES client: %s", err)
	}

	ES = client
	log.Println("Connected to Elasticsearch successfully")
}
