package api

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/RichardHoa/hack-me/internal/constants"
	"github.com/RichardHoa/hack-me/internal/store"
	"github.com/RichardHoa/hack-me/internal/utils"
)

type ChatboxHandler struct {
	Logger   *log.Logger
	VectorDB store.VectorDB
	AI       store.AIClient
}

func NewChatboxHandler(logger *log.Logger, ai store.AIClient, vdb store.VectorDB) *ChatboxHandler {
	return &ChatboxHandler{
		Logger:   logger,
		AI:       ai,
		VectorDB: vdb,
	}
}

type ChatHistoryItem struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Question string            `json:"question"`
	History  []ChatHistoryItem `json:"history,omitempty"`
}

type RawDoc struct {
	URL   string `json:"url"`
	Title string `json:"title"`
	Text  string `json:"text"`
}

type IngestRequest struct {
	Docs []RawDoc `json:"docs"`
}

type RelevantVectorDoc struct {
	Order string
	Title string
	URL   string
	Text  string
	Score float32
}

func (handler *ChatboxHandler) HandleChat(w http.ResponseWriter, r *http.Request) {
	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handler.Logger.Printf("ERROR: HandleChat > json encoding: %v", err)
		utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage(constants.StatusInvalidJSONMessage, constants.MSG_MALFORMED_REQUEST_DATA, "request"))
		return
	}

	trimmedQuestion := strings.TrimSpace(req.Question)
	if trimmedQuestion == "" {
		utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage(constants.StatusInvalidJSONMessage, constants.MSG_LACKING_MANDATORY_FIELDS, "question"))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	var hist strings.Builder
	for _, hisItem := range req.History {
		role := strings.ToLower(strings.TrimSpace(hisItem.Role))
		content := strings.TrimSpace(hisItem.Content)
		if content == "" {
			continue
		}
		switch role {
		case "assistant":
			hist.WriteString("Assistant: ")
		case "system":
			hist.WriteString("System: ")
		default:
			hist.WriteString("User: ")
		}
		hist.WriteString(content)
		hist.WriteString("\n")
	}

	docs, err := handler.SearchRelevantDoc(ctx, trimmedQuestion, 5)
	if err != nil {
		handler.Logger.Printf("RAG retrieve error: %v\n", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.NewMessage(
			constants.StatusInternalErrorMessage, "", ""))
		return
	}

	var contextBuilder strings.Builder
	total := 0
	for _, item := range docs {
		seg := "### " + item.Title + " (" + item.URL + ")\n" + item.Text + "\n\n"
		if total+len(seg) > constants.MaxContextLength {
			handler.Logger.Printf("question %v retrieves too much docs %v, maxLength is %v\n", trimmedQuestion, len(docs), constants.MaxContextLength)
			break
		}
		contextBuilder.WriteString(seg)
		total += len(seg)
	}
	contextText := contextBuilder.String()

	userPrompt := "QUESTION:\n" + trimmedQuestion + "\n\nCONTEXT:\n" + contextText + "\n\nHISTORY:\n" + hist.String()

	handler.Logger.Println("---------- START AI CHAT ----------")
	handler.Logger.Println(userPrompt)

	answer, err := handler.AI.Generate(ctx, constants.SystemPrompts, userPrompt)
	if err != nil {
		handler.Logger.Printf("AI cannot generate content: %v\n", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.NewMessage(
			constants.StatusInternalErrorMessage, "", ""))
		return
	}

	handler.Logger.Println("---------- START AI ANSWER ----------")
	handler.Logger.Println(answer)
	handler.Logger.Println("---------- END AI CHAT ----------")
	utils.WriteJSON(w, http.StatusOK, utils.Message{
		"data": []map[string]any{
			{"response": strings.TrimSpace(answer)},
		},
	})
}

func (handler *ChatboxHandler) AddDocsToVectorDB(w http.ResponseWriter, r *http.Request) {
	var req IngestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage(
			constants.StatusInvalidJSONMessage, constants.MSG_MALFORMED_REQUEST_DATA, "request"))
		return
	}
	if len(req.Docs) == 0 {
		utils.WriteJSON(w, http.StatusBadRequest, utils.NewMessage(
			constants.StatusInvalidJSONMessage, constants.MSG_LACKING_MANDATORY_FIELDS, "docs"))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	var batch []store.VectorPoint
	intVectorDimensions := int32(constants.VectorDimensions)

	for _, doc := range req.Docs {
		if err := utils.ValidateJSONFieldsNotEmpty(w, req); err != nil {
			return
		}
		chunks := SplitTextIntoChunks(doc.Text, 1000, 200)

		for i, chunk := range chunks {
			embedding, err := handler.AI.EmbedText(ctx, chunk, intVectorDimensions)
			if err != nil {
				handler.Logger.Printf("Cannot embed chunk, error: %v, chunk: %v\n", err, chunk)
				utils.WriteJSON(w, http.StatusInternalServerError, utils.NewMessage(
					constants.StatusInternalErrorMessage, "", ""))
				return
			}
			if len(embedding) != constants.VectorDimensions {
				handler.Logger.Printf("unexpected embedding dimensions: %d", len(embedding))
				utils.WriteJSON(w, http.StatusInternalServerError, utils.NewMessage(
					constants.StatusInternalErrorMessage, "", ""))
				return
			}

			idNumber := hashToUint64(fmt.Sprintf("%s|%d|%s", doc.URL, i, chunk))
			batch = append(batch, store.VectorPoint{
				ID:     idNumber,
				Vector: embedding,
				Payload: map[string]any{
					"url":   doc.URL,
					"title": doc.Title,
					"ord":   i,
					"text":  chunk,
				},
			})

			// flush in batches
			if len(batch) >= 128 {
				if err := handler.VectorDB.Upsert(ctx, batch); err != nil {
					utils.WriteJSON(w, http.StatusInternalServerError, utils.NewMessage(
						constants.StatusInternalErrorMessage, "vector upsert failed", doc.URL))
					return
				}
				batch = batch[:0]
			}
		}
	}

	if len(batch) > 0 {
		if err := handler.VectorDB.Upsert(ctx, batch); err != nil {
			handler.Logger.Printf("vector upsert failed: %v", err)
			utils.WriteJSON(w, http.StatusInternalServerError, utils.NewMessage(
				constants.StatusInternalErrorMessage, "", ""))
			return
		}
	}

	utils.WriteJSON(w, http.StatusCreated, utils.Message{"message": "add data successfully"})
}

func (handler *ChatboxHandler) SearchRelevantDoc(ctx context.Context, question string, limit uint64) ([]RelevantVectorDoc, error) {
	vector, err := handler.AI.EmbedText(ctx, question, 0)
	if err != nil {
		return nil, fmt.Errorf("embed question %v failed: %w", question, err)
	}

	points, err := handler.VectorDB.Query(ctx, vector, limit, []string{"title", "url", "text", "ord"})
	if err != nil {
		return nil, fmt.Errorf("search fail: %v", err)
	}

	out := make([]RelevantVectorDoc, 0, len(points))
	for _, p := range points {
		title, ok1 := p.Payload["title"].(string)
		url, ok2 := p.Payload["url"].(string)
		text, ok3 := p.Payload["text"].(string)
		ordVal, ok4 := p.Payload["ord"]

		if !ok1 || !ok2 || !ok3 || !ok4 {
			return nil, fmt.Errorf("payload missing required fields: %+v", p.Payload)
		}

		var orderStr string
		switch v := ordVal.(type) {
		case string:
			orderStr = v
		case int:
			orderStr = strconv.Itoa(v)
		case int64:
			orderStr = strconv.FormatInt(v, 10)
		case uint64:
			orderStr = strconv.FormatUint(v, 10)
		case float64:
			// JSON decoding often gives numbers as float64
			orderStr = strconv.FormatFloat(v, 'f', -1, 64)
		default:
			return nil, fmt.Errorf("ord has unsupported type %T", v)
		}

		out = append(out, RelevantVectorDoc{
			Title: title,
			URL:   url,
			Text:  text,
			Order: orderStr,
			Score: p.Score,
		})
	}

	return out, nil
}

func SplitTextIntoChunks(text string, maxSize, overlapSize int) []string {
	paragraphs := strings.Split(strings.TrimSpace(text), "\n\n")
	var chunks []string
	var current strings.Builder

	for _, p := range paragraphs {
		if current.Len()+len(p)+2 > maxSize && current.Len() > 0 {
			chunk := strings.TrimSpace(current.String())
			if chunk != "" {
				chunks = append(chunks, chunk)
			}
			current.Reset()
			if len(chunk) > overlapSize {
				tail := chunk[len(chunk)-overlapSize:]
				current.WriteString(tail)
				current.WriteString("\n\n")
			}
		}
		current.WriteString(p)
		current.WriteString("\n\n")
	}
	if leftover := strings.TrimSpace(current.String()); leftover != "" {
		chunks = append(chunks, leftover)
	}
	return chunks
}

func hashToUint64(s string) uint64 {
	sum := sha256.Sum256([]byte(s))
	var out uint64
	for i := 0; i < 4; i++ {
		part := binary.LittleEndian.Uint64(sum[i*8 : (i+1)*8])
		out ^= part
	}
	return out
}
