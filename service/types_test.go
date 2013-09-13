package service

import (
	"encoding/json"
	"log"
	"net/url"
	"testing"
)

func TestControlAPI(t *testing.T) {

	mysqlUri, err := url.Parse("tcp://0.0.0.0:3306")
	if err != nil {
		t.Fatal("Error parsing mysqlUri: ", err)
	}

	conn := Connection{
		Uri:          *mysqlUri,
		Application:  "mysql",
		Relationship: CONN_LISTENS,
	}
	out, err := json.Marshal(conn)
	if err != nil {
		t.Fatal("Unexpected Marshal error on mysql Connection: ", err)
	}
	var connImported Connection
	err = json.Unmarshal(out, &connImported)
	if err != nil {
		t.Fatal("Unexpected Unmarshal error on mysql Connection: ", err)
	}
	log.Print(string(out))

}
