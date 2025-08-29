package api

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/RichardHoa/hack-me/internal/constants"
	"github.com/RichardHoa/hack-me/internal/utils"
	"google.golang.org/genai"

	"github.com/qdrant/go-client/qdrant"
)

type ChatboxHandler struct {
	Logger       *log.Logger
	QdrantClient *qdrant.Client
	GeminiClient *genai.Client
}

func NewChatboxHandler(logger *log.Logger, AIClient *genai.Client, VectorClient *qdrant.Client) *ChatboxHandler {
	return &ChatboxHandler{
		Logger:       logger,
		GeminiClient: AIClient,
		QdrantClient: VectorClient,
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

	hist.WriteString("User: ")
	hist.WriteString(trimmedQuestion)

	docs, err := handler.SearchRelevantDoc(ctx, trimmedQuestion, 10)
	if err != nil {
		handler.Logger.Printf("RAG retrieve error: %v\n", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.NewMessage(
			constants.StatusInternalErrorMessage, "", ""))
		return
	}

	var contextBuilder strings.Builder
	total := 0
	for _, item := range docs {
		handler.Logger.Printf("LOGGING: text %s has score %v\n", item.Title, item.Score)

		seg := "### " + item.Title + " (" + item.URL + ")\n" + item.Text + "\n\n"
		if total+len(seg) > constants.MaxContextLength {
			handler.Logger.Printf("question %v retrieves too much docs %v\n", trimmedQuestion, docs)
			break
		}
		contextBuilder.WriteString(seg)
		total += len(seg)
	}
	contextText := contextBuilder.String()

	userPrompt := "QUESTION:\n" + trimmedQuestion + "\n\nCONTEXT:\n" + contextText + "\n\nHISTORY:\n" + hist.String()

	result, err := handler.GeminiClient.Models.GenerateContent(
		ctx,
		constants.AIModel,
		genai.Text(constants.SystemPrompts+"\n\n"+userPrompt),
		nil,
	)
	if err != nil {
		handler.Logger.Printf("AI cannot generate content: %v\n", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.NewMessage(
			constants.StatusInternalErrorMessage, "", ""))
		return
	}

	answer := strings.TrimSpace(result.Text())

	utils.WriteJSON(w, http.StatusOK, utils.Message{
		"data": []map[string]any{
			{"response": answer},
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

	var batch []*qdrant.PointStruct

	intVectorDimensions := int32(constants.VectorDimensions)

	for _, doc := range req.Docs {
		err := utils.ValidateJSONFieldsNotEmpty(w, req)
		if err != nil {
			return
		}

		chunks := SplitTextIntoChunks(doc.Text, 1000, 200)

		for i, chunk := range chunks {
			embededResponse, err := handler.GeminiClient.Models.EmbedContent(
				ctx,
				"text-embedding-004",
				genai.Text(chunk),
				&genai.EmbedContentConfig{
					OutputDimensionality: &intVectorDimensions,
				},
			)

			if err != nil {
				handler.Logger.Printf("Cannot embed chunk, error: %v, chunk: %v\n", err, chunk)
				utils.WriteJSON(w, http.StatusInternalServerError, utils.NewMessage(
					constants.StatusInternalErrorMessage, "", ""))
				return
			}

			if len(embededResponse.Embeddings) == 0 {
				handler.Logger.Printf("Empty embedded response")
				utils.WriteJSON(w, http.StatusInternalServerError, utils.NewMessage(
					constants.StatusInternalErrorMessage, "", ""))
				return
			}

			embeddingValues := embededResponse.Embeddings[0].Values

			vector := make([]float32, len(embeddingValues))
			copy(vector, embeddingValues)

			if len(vector) != constants.VectorDimensions {
				handler.Logger.Printf("unexpected embedding dimmensions: %d", len(vector))
				utils.WriteJSON(w, http.StatusInternalServerError, utils.NewMessage(
					constants.StatusInternalErrorMessage, "", ""))
				return
			}

			idNumber := hashToUint64(fmt.Sprintf("%s|%d|%s", doc.URL, i, chunk))

			point := &qdrant.PointStruct{
				Id: &qdrant.PointId{
					PointIdOptions: &qdrant.PointId_Num{
						Num: idNumber,
					},
				},
				Vectors: &qdrant.Vectors{
					VectorsOptions: &qdrant.Vectors_Vector{
						Vector: &qdrant.Vector{Data: vector},
					},
				},
				Payload: qdrant.NewValueMap(map[string]any{
					"url":   doc.URL,
					"title": doc.Title,
					"ord":   i,
					"text":  chunk,
				}),
			}
			batch = append(batch, point)

			// 3) flush in batches
			if len(batch) >= 128 {
				if err := upsertBatch(ctx, handler.QdrantClient, constants.VectorCollectionName, batch); err != nil {
					utils.WriteJSON(w, http.StatusInternalServerError, utils.NewMessage(
						constants.StatusInternalErrorMessage, "qdrant upsert failed", doc.URL))
					return
				}
				batch = batch[:0]
			}
		}
	}
	if len(batch) > 0 {
		err := upsertBatch(ctx, handler.QdrantClient, constants.VectorCollectionName, batch)
		if err != nil {
			handler.Logger.Printf("qdrant upsert failed: %v", err)
			utils.WriteJSON(w, http.StatusInternalServerError, utils.NewMessage(
				constants.StatusInternalErrorMessage, "", ""))
			return
		}
	}

	utils.WriteJSON(w, http.StatusCreated, utils.Message{"message": "add data successfully"})
}

func (handler *ChatboxHandler) SearchRelevantDoc(ctx context.Context, question string, limit uint64) ([]RelevantVectorDoc, error) {
	embededResponse, err := handler.GeminiClient.Models.EmbedContent(ctx, "text-embedding-004", genai.Text(question), nil)
	if err != nil {
		return nil, fmt.Errorf("embed question %v failed: %w", question, err)
	}

	if len(embededResponse.Embeddings) == 0 {
		return nil, fmt.Errorf("empty embedding")
	}

	vector := embededResponse.Embeddings[0].Values

	points, err := handler.QdrantClient.Query(ctx, &qdrant.QueryPoints{
		CollectionName: constants.VectorCollectionName,
		Query:          qdrant.NewQuery(vector...),
		Limit:          &limit,
		WithPayload:    qdrant.NewWithPayloadInclude("title", "url", "text"),
	})
	if err != nil {
		return nil, fmt.Errorf("Search fail: %v", err)
	}

	out := make([]RelevantVectorDoc, 0, len(points))
	for _, point := range points {
		payload := point.Payload
		out = append(out, RelevantVectorDoc{
			Title: payload["title"].GetStringValue(),
			URL:   payload["url"].GetStringValue(),
			Text:  payload["text"].GetStringValue(),
			Score: float32(point.Score),
		})
	}

	return out, nil
}

func upsertBatch(ctx context.Context, cli *qdrant.Client, col string, pts []*qdrant.PointStruct) error {
	wait := true
	_, err := cli.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: col,
		Points:         pts,
		Wait:           &wait,
	})
	if err != nil {
		return fmt.Errorf("QDRANT UPSERT ERROR: %v", err)
	}

	return nil
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
