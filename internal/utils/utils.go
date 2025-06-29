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
func NewMessage(message, errCode, errSource string) Message {
	createdMessage := Message{"message": message}

	if errCode != "" && errSource != "" {
		createdMessage["errors"] = []Message{
			{
				"source": errSource,
				"code":   errCode,
			},
		}
	}

	return createdMessage
}

func WriteJSON(w http.ResponseWriter, statusCode int, data Message) error {
	jsonBytes, err := json.MarshalIndent(data, "", " ")
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Language", "en-US")
	w.WriteHeader(statusCode)

	w.Write(jsonBytes)

	return nil
}

func NullIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
