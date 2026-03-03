package elastic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
)

func FindLatestByRack(
	rackName string,
	pipelineName string,
	operationalCategory string,
) (string, map[string]interface{}, error) {

	log.Println("Searching index:", IndexName)
	log.Println("Rack:", rackName)
	log.Println("Pipeline:", pipelineName)
	log.Println("Category:", operationalCategory)

	query := map[string]interface{}{
		"size": 1,
		"sort": []map[string]interface{}{
			{
				"timestamp": map[string]interface{}{
					"order": "desc",
				},
			},
		},
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []map[string]interface{}{

					{
						"term": map[string]interface{}{
							"rack_name": rackName,
						},
					},

					{
						"term": map[string]interface{}{
							"pipeline_name": pipelineName,
						},
					},

					{
						"term": map[string]interface{}{
							"operational_category": operationalCategory,
						},
					},
					{
						"term": map[string]interface{}{
							"job_status": "FAILED",
						},
					},
				},
			},
		},
	}

	data, err := json.Marshal(query)
	if err != nil {
		return "", nil, err
	}

	res, err := ES.Search(
		ES.Search.WithContext(context.Background()),
		ES.Search.WithIndex(IndexName),
		ES.Search.WithBody(bytes.NewReader(data)),
	)

	if err != nil {
		return "", nil, err
	}

	defer res.Body.Close()

	var result map[string]interface{}

	err = json.NewDecoder(res.Body).Decode(&result)
	if err != nil {
		return "", nil, err
	}

	hits := result["hits"].(map[string]interface{})["hits"].([]interface{})

	if len(hits) == 0 {
		log.Println("No matching document found")
		return "", nil, fmt.Errorf("no existing build found")
	}

	hit := hits[0].(map[string]interface{})

	docID := hit["_id"].(string)
	source := hit["_source"].(map[string]interface{})

	log.Println("Found document ID:", docID)

	return docID, source, nil
}

// --------------------------

func FindByBuildID(buildID string) (string, map[string]interface{}, error) {

	query := map[string]interface{}{
		"size": 1,
		"query": map[string]interface{}{
			"term": map[string]interface{}{
				"build_id.keyword": buildID,
			},
		},
	}

	data, _ := json.Marshal(query)

	res, err := ES.Search(
		ES.Search.WithContext(context.Background()),
		ES.Search.WithIndex(IndexName),
		ES.Search.WithBody(bytes.NewReader(data)),
	)

	if err != nil {
		return "", nil, err
	}

	defer res.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(res.Body).Decode(&result)

	hits := result["hits"].(map[string]interface{})["hits"].([]interface{})

	if len(hits) == 0 {
		return "", nil, fmt.Errorf("build not found")
	}

	hit := hits[0].(map[string]interface{})

	return hit["_id"].(string),
		hit["_source"].(map[string]interface{}),
		nil
}

func GetDocumentByID(docID string) (map[string]interface{}, error) {

	res, err := ES.Get(
		IndexName,
		docID,
		ES.Get.WithContext(context.Background()),
	)

	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("document not found")
	}

	var result map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, err
	}

	source := result["_source"].(map[string]interface{})
	return source, nil
}
