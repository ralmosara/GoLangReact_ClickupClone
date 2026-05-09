package httpx

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"

	"github.com/yourorg/clickup/internal/observability"
)

func JSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if body == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(body)
}

func Err(w http.ResponseWriter, code int, msg string) {
	JSON(w, code, map[string]string{"error": msg})
}

// Fail logs the underlying error via the request-scoped logger (so
// request_id, method, path are attached automatically) and writes a
// sanitized client response. Use this for 5xx and any other path where the
// raw err may contain internal detail (driver text, file paths, wrapped
// service errors) — the underlying err is logged but never returned to the
// caller.
func Fail(w http.ResponseWriter, r *http.Request, code int, publicMsg string, err error) {
	if err != nil {
		observability.FromContext(r.Context()).Error("request failed",
			"code", code,
			"err", err,
		)
	}
	JSON(w, code, map[string]string{"error": publicMsg})
}

func Decode(r *http.Request, v any) error {
	if r.Body == nil {
		return errors.New("empty body")
	}
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

func ParseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}
