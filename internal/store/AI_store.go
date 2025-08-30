package store

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/RichardHoa/hack-me/internal/constants"
	"github.com/qdrant/go-client/qdrant"
	"google.golang.org/genai"
)

type AIClient interface {
	EmbedText(ctx context.Context, text string, dims int32) ([]float32, error)
	Generate(ctx context.Context, systemPrompt, userPrompt string) (string, error)
}

type VectorDB interface {
	Upsert(ctx context.Context, points []VectorPoint) error
	Query(ctx context.Context, vector []float32, limit uint64, withPayload []string) ([]VectorPoint, error)
}

type VectorPoint struct {
	ID      any
	Vector  []float32
	Payload map[string]any
	Score   float32
}

type GeminiAI struct {
	Client               *genai.Client
	GenModel             string
	EmbedModel           string
	DefaultOutDimensions int32
}

func InitAI(ctx context.Context) (*GeminiAI, error) {
	key := constants.AISecretKey

	genAIClient, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  key,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return &GeminiAI{}, err
	}
	return &GeminiAI{
		Client:               genAIClient,
		GenModel:             constants.AIModel,
		EmbedModel:           constants.AIEmbededModel,
		DefaultOutDimensions: constants.VectorDimensions,
	}, nil
}

func InitVectorDB(ctx context.Context) (*QdrantDB, error) {
	intVectorPort, err := strconv.Atoi(constants.VectorPort)
	if err != nil {
		return &QdrantDB{}, fmt.Errorf("Qdrant port is not valid integer %d", intVectorPort)
	}

	client, err := qdrant.NewClient(&qdrant.Config{
		Host:   constants.VectorHost,
		Port:   intVectorPort,
		APIKey: constants.VectorSecret,
		UseTLS: true,
	})
	if err != nil {
		return &QdrantDB{}, fmt.Errorf("qdrant: %w", err)
	}
	return &QdrantDB{
		Client:         client,
		CollectionName: constants.VectorCollectionName,
	}, nil
}

func (g *GeminiAI) EmbedText(ctx context.Context, text string, dims int32) ([]float32, error) {
	var outDims *int32
	switch {
	case dims > 0:
		outDims = &dims
	case g.DefaultOutDimensions > 0:
		outDims = &g.DefaultOutDimensions
	}
	resp, err := g.Client.Models.EmbedContent(ctx, g.EmbedModel, genai.Text(text), &genai.EmbedContentConfig{
		OutputDimensionality: outDims,
	})
	if err != nil {
		return nil, fmt.Errorf("gemini embed: %w", err)
	}
	if len(resp.Embeddings) == 0 {
		return nil, fmt.Errorf("gemini embed: empty embeddings")
	}
	v := make([]float32, len(resp.Embeddings[0].Values))
	copy(v, resp.Embeddings[0].Values)
	return v, nil
}

func (g *GeminiAI) Generate(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	resp, err := g.Client.Models.GenerateContent(ctx, g.GenModel, genai.Text(systemPrompt+"\n\n"+userPrompt), nil)
	if err != nil {
		return "", fmt.Errorf("gemini generate: %w", err)
	}
	return strings.TrimSpace(resp.Text()), nil
}

// QdrantDB adapts github.com/qdrant/go-client to VectorDB.
type QdrantDB struct {
	Client         *qdrant.Client
	CollectionName string
}

func (q *QdrantDB) Upsert(ctx context.Context, points []VectorPoint) error {
	if len(points) == 0 {
		return nil
	}
	pts := make([]*qdrant.PointStruct, 0, len(points))
	for _, p := range points {
		var id *qdrant.PointId
		switch v := p.ID.(type) {
		case uint64:
			id = &qdrant.PointId{PointIdOptions: &qdrant.PointId_Num{Num: v}}
		case string:
			id = &qdrant.PointId{PointIdOptions: &qdrant.PointId_Uuid{Uuid: v}}
		default:
			return fmt.Errorf("qdrant upsert: unsupported id type %T", p.ID)
		}
		pts = append(pts, &qdrant.PointStruct{
			Id: id,
			Vectors: &qdrant.Vectors{
				VectorsOptions: &qdrant.Vectors_Vector{
					Vector: &qdrant.Vector{Data: p.Vector},
				},
			},
			Payload: qdrant.NewValueMap(p.Payload),
		})
	}
	wait := true
	_, err := q.Client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: q.CollectionName,
		Points:         pts,
		Wait:           &wait,
	})
	if err != nil {
		return fmt.Errorf("qdrant upsert: %w", err)
	}
	return nil
}

func (q *QdrantDB) Query(ctx context.Context, vector []float32, limit uint64, withPayload []string) ([]VectorPoint, error) {
	resp, err := q.Client.Query(ctx, &qdrant.QueryPoints{
		CollectionName: q.CollectionName,
		Query:          qdrant.NewQuery(vector...),
		Limit:          &limit,
		WithPayload:    qdrant.NewWithPayloadInclude(withPayload...),
	})
	if err != nil {
		return nil, fmt.Errorf("qdrant query: %w", err)
	}
	out := make([]VectorPoint, 0, len(resp))
	for _, p := range resp {
		payload := map[string]any{}
		// qdrant.Value => unwrap
		for k, v := range p.Payload {
			switch vv := v.Kind.(type) {
			case *qdrant.Value_StringValue:
				payload[k] = vv.StringValue
			case *qdrant.Value_IntegerValue:
				payload[k] = vv.IntegerValue
			case *qdrant.Value_BoolValue:
				payload[k] = vv.BoolValue
			case *qdrant.Value_DoubleValue:
				payload[k] = vv.DoubleValue
			case *qdrant.Value_ListValue:
				payload[k] = vv.ListValue
			case *qdrant.Value_NullValue:
				payload[k] = nil
			default:
				payload[k] = v // fallback
			}
		}
		out = append(out, VectorPoint{
			ID:      p.Id.GetNum(),
			Vector:  nil,
			Payload: payload,
			Score:   float32(p.Score),
		})
	}
	return out, nil
}
