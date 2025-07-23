package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/RichardHoa/hack-me/internal/constants"
)

/*
Message defines a generic type for creating simple key-value JSON objects,
used for API responses.
*/
type Message map[string]any

/*
BeautifyJSON marshals an interface into a pretty-printed JSON string.
It returns an empty string if a marshaling error occurs. This function is
primarily intended for logging or debugging purposes.
*/
func BeautifyJSON(v any) string {
	bytes, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling JSON: %v\n", err)
		return ""
	}
	return (string(bytes))
}

/*
NewMessage creates a standard Message object for API responses.
If errCode and errSource are provided, it includes a structured error object.
*/
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

/*
WriteJSON is a helper function to send a JSON response. It marshals the
provided data, sets the appropriate headers (Content-Type, Content-Language),
writes the HTTP status code, and sends the JSON payload.
*/
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

/*
ValidateJSONFieldsNotEmpty uses reflection to check that all string fields of a
given struct with a 'json' tag are not empty or consist only of whitespace.
If an empty field is found, it automatically writes a 400 Bad Request error
response and returns an error. It's designed to be used for validating
API request bodies.
*/
func ValidateJSONFieldsNotEmpty(w http.ResponseWriter, input any) error {
	v := reflect.ValueOf(input)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}

		val := v.Field(i)
		if val.Kind() == reflect.String {
			trimmed := strings.TrimSpace(val.String())
			if trimmed == "" {
				WriteJSON(w, http.StatusBadRequest, NewMessage(
					fmt.Sprintf("field '%s' is required and cannot be empty or only whitespace", jsonTag),
					constants.MSG_LACKING_MANDATORY_FIELDS,
					jsonTag,
				))
				return errors.New("lacking required field")
			}
		}
	}

	return nil
}

/*
NullIfEmpty returns nil if the input string is empty or contains only
whitespace; otherwise, it returns the original string. This is useful for
preparing data for database insertion where an empty string should be
represented as NULL.
*/
func NullIfEmpty(s string) any {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return nil
	}
	return s
}
