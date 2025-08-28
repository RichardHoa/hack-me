package api

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
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

type chatHistoryItem struct {
	Role    string `json:"role"`    // optional: "user" | "assistant" | "system"
	Content string `json:"content"` // previous utterance
}

type chatboxReq struct {
	Question string            `json:"question"`
	History  []chatHistoryItem `json:"history,omitempty"`
}

type IngestDoc struct {
	URL   string `json:"url"`
	Title string `json:"title"`
	Text  string `json:"text"`
}
type ingestReq struct {
	Docs []IngestDoc `json:"docs"`
}

func (handler *ChatboxHandler) ReplyChat(w http.ResponseWriter, r *http.Request) {
	var req chatboxReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handler.Logger.Printf("ERROR: ReplyChat > json encoding: %v", err)
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

	// ------- (A) build HISTORY exactly as before -------
	var hist strings.Builder
	for _, t := range req.History {
		role := strings.ToLower(strings.TrimSpace(t.Role))
		content := strings.TrimSpace(t.Content)
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

	// ------- (B) RAG: fetch top-K chunks from Qdrant -------
	chunks, err := handler.retrieveTopK(ctx, trimmedQuestion, 3) // your function that does: embed -> qdrant.Query(...)
	if err != nil {
		handler.Logger.Printf("RAG retrieve error: %v", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.NewMessage(
			constants.StatusInternalErrorMessage, "retrieval failed", "vectordb"))
		return
	}

	// Build a tight CONTEXT (cap characters & number of chunks)
	var contextBuilder strings.Builder
	total := 0
	for _, c := range chunks {
		handler.Logger.Printf("text %s has score %v\n", c.Title, c.Score)
		seg := "### " + c.Title + " (" + c.URL + ")\n" + c.Text + "\n\n"
		if total+len(seg) > 12000 { // keep the prompt lean
			break
		}
		contextBuilder.WriteString(seg)
		total += len(seg)
	}
	contextText := contextBuilder.String()

	// ------- (C) Compose final prompt -------
	sys := `
		ROLE
		You are the website assistant for this application. You help users find accurate information about our site, products, features, pricing, policies, setup, and troubleshooting.

		PRIMARY DIRECTIVE (CONTEXT-FIRST RAG)
		1) You are given CONTEXT (retrieved snippets from our indexed content).
		2) If the answer is covered by CONTEXT, rely on it heavily and quote facts from it.
		3) If CONTEXT does not contain the required information, explicitly warn the user:
			"I don’t have this in my indexed content."
			Then, provide a best-effort general answer from your broader knowledge, but DO NOT invent site-specific facts (pricing, SKUs, policies, SLAs, release dates, emails, phone numbers) that are not present in CONTEXT. Where appropriate, recommend where the user might find the info (e.g., docs page, support, sales).

		CITATIONS
		- For any statement derived from CONTEXT, cite inline as [Title](URL) right after the sentence or bullet.
		- When no CONTEXT was used, include a line at the end: "Sources: none (general knowledge)".

		STYLE & UX
		- Be concise but helpful. Prefer short paragraphs or bullet points.
		- Use the user’s terminology. Define jargon briefly if it helps.
		- If the question is ambiguous or missing key details, ask at most ONE focused follow-up question.
		- If the user asks for a step-by-step, provide a numbered list.
		- If the user asks for comparisons or pros/cons, provide a tidy table or bullets.
		- If dates, units, versions, or limits are relevant, be explicit and concrete.

		SAFETY & NON-HALLUCINATION
		- Never fabricate internal links, SKUs, coupon codes, emails, or phone numbers. If not in CONTEXT, say you don’t have it in the indexed content.
		- Do not contradict CONTEXT. If HISTORY conflicts with CONTEXT, prefer CONTEXT.

		HISTORY
		- HISTORY may include previous turns; it exists to keep continuity (what the user already told us).
		- Never invent memory; only use what’s in HISTORY and CONTEXT.

		OUTPUT FORMAT
		- Answer text first.
		- If any CONTEXT was used, add a final "Sources:" section listing each [Title](URL) on its own line.
		- Keep the whole answer tightly scoped to the user’s question.

		EXAMPLES OF REQUIRED WARNINGS
		- "I don’t have this in my indexed content. Here’s a general overview…" (then provide helpful, non-site-specific guidance).
		- "I can’t find pricing details in my indexed content; please check the Pricing page or contact support."
	`

	// Later, you concatenate this with your built user prompt:
	// genai.Text(sys + "\n\n" + user)

	user := "QUESTION:\n" + trimmedQuestion + "\n\nCONTEXT:\n" + contextText + "\n\nHISTORY:\n" + hist.String()

	// ------- (D) Generate with Gemini as before -------
	result, err := handler.GeminiClient.Models.GenerateContent(
		ctx,
		constants.AIModel, // e.g., "gemini-2.5-flash"
		genai.Text(sys+"\n\n"+user),
		nil,
	)
	if err != nil {
		utils.WriteJSON(w, http.StatusInternalServerError, utils.NewMessage(
			constants.StatusInternalErrorMessage, "generation failed", "ai"))
		return
	}

	answer := strings.TrimSpace(result.Text())

	// ------- (E) Return in your existing shape -------
	utils.WriteJSON(w, http.StatusOK, utils.Message{
		"data": []map[string]any{
			{"response": answer},
		},
	})
}

func (h *ChatboxHandler) Ingest(w http.ResponseWriter, r *http.Request) {
	var req ingestReq
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

	intVectorDim := int32(constants.VectorDim)

	for _, d := range req.Docs {
		if d.URL == "" || d.Title == "" || strings.TrimSpace(d.Text) == "" {
			continue
		}
		chunks := chunkText(d.Text, 1000, 150) // ~1000 chars, 150 overlap

		for i, c := range chunks {
			// 1) embed with Gemini
			emb, err := h.GeminiClient.Models.EmbedContent(
				ctx,
				"text-embedding-004",
				genai.Text(c),
				&genai.EmbedContentConfig{
					OutputDimensionality: &intVectorDim,
				},
			)
			if err != nil {
				utils.WriteJSON(w, http.StatusInternalServerError, utils.NewMessage(
					constants.StatusInternalErrorMessage, "embedding failed", d.URL))
				return
			}
			if len(emb.Embeddings) == 0 {
				utils.WriteJSON(w, http.StatusInternalServerError, utils.NewMessage(
					constants.StatusInternalErrorMessage, "empty embedding", d.URL))
				return
			}
			v := emb.Embeddings[0].Values // []float32

			// (optional) copy if you plan to reuse v later
			vec := make([]float32, len(v))
			copy(vec, v)

			if len(vec) != constants.VectorDim {
				// log to catch mismatches early
				h.Logger.Printf("unexpected embedding dim: %d", len(vec))
			}

			// 2) build point payload
			idNum := hashToUint64(fmt.Sprintf("%s|%s|%d", d.URL, d.Title, i))

			point := &qdrant.PointStruct{
				Id: &qdrant.PointId{
					PointIdOptions: &qdrant.PointId_Num{
						Num: idNum,
					},
				},
				Vectors: &qdrant.Vectors{
					VectorsOptions: &qdrant.Vectors_Vector{
						Vector: &qdrant.Vector{Data: vec},
					},
				},
				Payload: qdrant.NewValueMap(map[string]any{
					"url":   d.URL,
					"title": d.Title,
					"ord":   i,
					"text":  c,
				}),
			}
			batch = append(batch, point)

			// 3) flush in batches
			if len(batch) >= 128 {
				if err := upsertBatch(ctx, h.QdrantClient, constants.VectorCollectionName, batch); err != nil {
					utils.WriteJSON(w, http.StatusInternalServerError, utils.NewMessage(
						constants.StatusInternalErrorMessage, "qdrant upsert failed", d.URL))
					return
				}
				batch = batch[:0]
			}
		}
	}
	if len(batch) > 0 {
		if err := upsertBatch(ctx, h.QdrantClient, constants.VectorCollectionName, batch); err != nil {
			utils.WriteJSON(w, http.StatusInternalServerError, utils.NewMessage(
				constants.StatusInternalErrorMessage, "qdrant upsert failed", "batch"))
			return
		}
	}

	utils.WriteJSON(w, http.StatusOK, utils.Message{"data": []string{"ok"}})
}

func upsertBatch(ctx context.Context, cli *qdrant.Client, col string, pts []*qdrant.PointStruct) error {
	wait := true
	_, err := cli.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: col,
		Points:         pts,
		Wait:           &wait, // *bool
	})
	if err != nil {
		log.Printf("QDRANT UPSERT ERROR: %v", err)
	}
	return err
}

func chunkText(s string, target, overlap int) []string {
	paras := strings.Split(strings.TrimSpace(s), "\n\n")
	var out []string
	var cur strings.Builder
	for _, p := range paras {
		if cur.Len()+len(p)+2 > target && cur.Len() > 0 {
			seg := strings.TrimSpace(cur.String())
			if seg != "" {
				out = append(out, seg)
			}
			// overlap
			if len(seg) > overlap {
				tail := seg[len(seg)-overlap:]
				cur.Reset()
				cur.WriteString(tail)
				cur.WriteString("\n\n")
			} else {
				cur.Reset()
			}
		}
		cur.WriteString(p)
		cur.WriteString("\n\n")
	}
	if x := strings.TrimSpace(cur.String()); x != "" {
		out = append(out, x)
	}
	return out
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func hashToUint64(s string) uint64 {
	sum := sha256.Sum256([]byte(s))
	return binary.LittleEndian.Uint64(sum[:8])
}

// retrieved chunk shape for local use
type ragChunk struct {
	Title string
	URL   string
	Text  string
	Score float32
}

// embed the query and search Qdrant
func (h *ChatboxHandler) retrieveTopK(ctx context.Context, question string, limit uint64) ([]ragChunk, error) {
	// A) embed question (same model you used at ingest)
	emb, err := h.GeminiClient.Models.EmbedContent(ctx, "text-embedding-004", genai.Text(strings.TrimSpace(question)), nil)
	if err != nil {
		return nil, fmt.Errorf("embed failed: %w", err)
	}
	if len(emb.Embeddings) == 0 {
		return nil, fmt.Errorf("empty embedding")
	}
	vec := emb.Embeddings[0].Values // []float32 (likely 768-dim)

	// B) search Qdrant
	points, err := h.QdrantClient.Query(ctx, &qdrant.QueryPoints{
		CollectionName: constants.VectorCollectionName,
		Query:          qdrant.NewQuery(vec...), // expand []float32
		Limit:          &limit,                  // *uint64
		WithPayload:    qdrant.NewWithPayloadInclude("title", "url", "text"),
	})
	if err != nil {
		h.Logger.Printf("Qdrant query error: %v", err)
		return nil, fmt.Errorf("Search fail")
	}

	// Map results into your ragChunk
	out := make([]ragChunk, 0, len(points))
	for _, p := range points {
		pl := p.Payload
		out = append(out, ragChunk{
			Title: pl["title"].GetStringValue(),
			URL:   pl["url"].GetStringValue(),
			Text:  pl["text"].GetStringValue(),
			Score: float32(p.Score),
		})
	}
	return out, nil
}

// build a bounded context block
func buildContext(chunks []ragChunk, maxChars int) string {
	var b strings.Builder
	total := 0
	for i, c := range chunks {
		seg := "### " + c.Title + " (" + c.URL + ")\n" + c.Text + "\n\n"
		if total+len(seg) > maxChars {
			break
		}
		b.WriteString(seg)
		total += len(seg)
		if i >= 4 { // at most ~5 chunks
			break
		}
	}
	return b.String()
}

func formatSources(chunks []ragChunk) []map[string]any {
	out := make([]map[string]any, 0, len(chunks))
	for _, c := range chunks {
		out = append(out, map[string]any{
			"title": c.Title,
			"url":   c.URL,
			"score": c.Score,
		})
	}
	return out
}
