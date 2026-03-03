package elastic

import (
	"bytes"
	"context"
	"encoding/json"
)

func UpdateDocument(docID string, document interface{}) error {

	data, err := json.Marshal(document)
	if err != nil {
		return err
	}

	res, err := ES.Index(
		IndexName,
		bytes.NewReader(data),
		ES.Index.WithDocumentID(docID),
		ES.Index.WithContext(context.Background()),
	)

	if err != nil {
		return err
	}

	defer res.Body.Close()

	return nil
}

// --------------------------------------------------
// Insert New Document
// --------------------------------------------------

func InsertDocumentWithID(docID string, document interface{}) error {

	data, err := json.Marshal(document)
	if err != nil {
		return err
	}

	res, err := ES.Index(
		IndexName,
		bytes.NewReader(data),
		ES.Index.WithDocumentID(docID),
		ES.Index.WithContext(context.Background()),
	)

	if err != nil {
		return err
	}

	defer res.Body.Close()

	return nil
}
