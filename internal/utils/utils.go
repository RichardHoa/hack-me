package utils

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type Message map[string]interface{}

func BeautifyJSON(v interface{}) string {
	bytes, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling JSON: %v\n", err)
		return ""
	}
	return (string(bytes))
}

func WriteJSON(w http.ResponseWriter, statusCode int, data Message) error {
	jsonBytes, err := json.MarshalIndent(data, "", " ")
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	w.Write(jsonBytes)

	return nil
}
