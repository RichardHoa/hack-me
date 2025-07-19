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

type Message map[string]any

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

func ValidateJSONFieldsNotEmpty(w http.ResponseWriter, input interface{}) error {
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

func NullIfEmpty(s string) interface{} {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return nil
	}
	return s
}
