/*
Package constants centralizes the management of application-wide constants and
environment-dependent configuration. It handles loading secrets and settings
from a .env file or the system environment, and provides a single source of
truth for values like token secrets, error codes, and standard messages.
*/
package constants

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

func LoadEnv() error {
	// Load env file in local dev
	if _, err := os.Stat(".env"); err == nil {
		err := godotenv.Load(".env")
		if err != nil {
			fmt.Printf("Error loading .env file: %v\n", err)
		}
	} else if _, err := os.Stat("../../.env"); err == nil {
		err := godotenv.Load("../../.env")
		if err != nil {
			fmt.Printf("Error loading ../../.env file: %v\n", err)
		}
	} else {
		fmt.Printf("No .env file found\n")
	}

	requiredSecrets := map[string]*string{
		"REFRESH_TOKEN_SECRET":   &RefreshTokenSecret,
		"ACCESS_TOKEN_SECRET":    &AccessTokenSecret,
		"CSRF_TOKEN_SECRET":      &CSRFTokenSecret,
		"AI_SECRET_KEY":          &AISecretKey,
		"VECTOR_DB_HOST":         &VectorHost,
		"VECTOR_DB_PORT":         &VectorPort,
		"VECTOR_DB_SECRET":       &VectorSecret,
		"VECTOR_COLLECTION_NAME": &VectorCollectionName,
	}

	var missing []string
	for key, dst := range requiredSecrets {
		val := os.Getenv(key)
		if val == "" {
			missing = append(missing, key)
		}
		*dst = val
	}

	if p := os.Getenv("APP_PORT"); p != "" {
		if n, err := strconv.Atoi(p); err == nil {
			AppPort = n
		} else {
			return fmt.Errorf("invalid APP_PORT %q: %w", p, err)
		}
	}

	IsDevMode = os.Getenv("DEV_MODE") == "LOCAL"

	if len(missing) > 0 {
		fmt.Println("--- DEBUG: Missing required secrets ---")
		for _, k := range missing {
			fmt.Printf("%s is missing\n", k)
		}
		fmt.Println("--- END DEBUG DUMP ---")
		return errors.New("missing required secrets: " + strings.Join(missing, ", "))
	}

	return nil
}

// Global variables holding configuration loaded from the environment.
var (
	RefreshTokenSecret   string
	AccessTokenSecret    string
	CSRFTokenSecret      string
	AppPort              int
	IsDevMode            bool
	AISecretKey          string
	VectorSecret         string
	VectorHost           string
	VectorPort           string
	VectorCollectionName string
)

// Defines the keys for standard claims within JSON Web Tokens.
const (
	JWTRefreshTokenID = "SomeThing"
	JWTUserName       = "userName"
	JWTUserID         = "someOtherThing"
)

// Defines constants for AI related functions.
const (
	AIModel          = "gemini-2.5-flash"
	AIEmbededModel   = "text-embedding-004"
	MaxContextLength = 12000
	SystemPrompts    = `
		ROLE
		You are the website assistant for this application. You help users find accurate information about our site, products, features, pricing, policies, setup, and troubleshooting.

		PRIMARY DIRECTIVE (CONTEXT-FIRST RAG)
		1) You are given CONTEXT (retrieved snippets from our indexed content).
		2) If the answer is covered by CONTEXT, rely on it heavily and quote facts from it.
		3) If CONTEXT does not contain the required information, explicitly warn the user:
			"I don’t have this information."
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
		- "I don’t have this in my content. Here’s a general overview…" (then provide helpful, non-site-specific guidance).
		- "I can’t find pricing details; please check the Pricing page or contact support."
		`
	VectorDimensions = 768
)

// Defines general application constants
const (
	StatusInternalErrorMessage = "Internal server error"
	StatusInvalidBodyMessage   = "Invalid request body > ERROR 100"
	UnauthorizedMessage        = "Unauthorized"
	ForbiddenMessage           = "Forbidden"
	MaxRequestBodySize         = 5 * 1024 * 1024 // 5MB
	AccessTokenTime            = 15 * time.Minute
	RefreshTokenTime           = 7 * (24 * time.Hour)
	CommentNestedLevel         = 5
	DefaultPageSize            = 10
	DefaultPage                = 1
)

// Defines standard error codes for API request validation failures.
const (
	MSG_INVALID_REQUEST_DATA     = "INVALID_REQUEST_DATA"
	MSG_MALFORMED_REQUEST_DATA   = "MALFORMED_REQUEST_DATA"
	MSG_CONFLICTING_FIELDS       = "CONFLICTING_FIELDS"
	MSG_LACKING_MANDATORY_FIELDS = "LACKING_MANDATORY_FIELDS"
)

// Defines enumerated integer codes for classifying various error types.
const (
	// Postgres Err
	PQUniqueViolation = iota
	PQForeignKeyViolation
	PQInvalidEnum
	PQInvalidTextRepresentation
	PQNotNullViolation
	PQCheckViolation
	PQNumericValueOutOfRange
	PQInvalidUUIDFormat
	PQDatatypeMismatch
	PQSyntaxError
	PQInvalidByteSequence
	// Application Err
	ResourceNotFound
	InvalidData
	InternalError
	LackingPermission
)

/*
PQErrorMessages maps standard PostgreSQL error codes (as strings) to the
application's internal integer-based error classification codes.
Source: https://www.postgresql.org/docs/current/errcodes-appendix.html
*/
var PQErrorMessages = map[string]int{
	"23505": PQUniqueViolation,
	"23503": PQForeignKeyViolation,
	"22P02": PQInvalidTextRepresentation,
	"23502": PQNotNullViolation,
	"23514": PQCheckViolation,
	"22003": PQNumericValueOutOfRange,
	"22P05": PQInvalidEnum,
	"22P04": PQInvalidUUIDFormat,
	"42804": PQDatatypeMismatch,
	"42601": PQSyntaxError,
	"22021": PQInvalidByteSequence, //input contains null bytes
}
