package elastic

import (
	"bytes"
	"encoding/json"
	"github.com/elastic/go-elasticsearch/v7/esapi"
)

func BuildSearchRequest(query interface{}, index string) (esapi.SearchRequest, error) {
	// Build the request body.
	var buf bytes.Buffer

	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		plog.Errorf("Error encoding query: %s", err)
		return esapi.SearchRequest{}, err
	}

	size := 10000
	version := true
	seqNoPrimaryTerm := true
	search := esapi.SearchRequest{
		Index:            []string{index},
		Body:             &buf,
		Size:             &size,
		Version:          &version,
		SeqNoPrimaryTerm: &seqNoPrimaryTerm,
	}

	return search, nil
}
