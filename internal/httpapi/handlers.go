package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"

	"task/internal/httpapi/middleware"
	"task/internal/model"
	"task/internal/service"
)

type handlers struct {
	app          *service.App
	logger       *slog.Logger
	maxBodyBytes int64
}

func decodeBody(w http.ResponseWriter, r *http.Request, dst any, maxBytes int64) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return newAPIError(http.StatusBadRequest, "invalid_request", "invalid request body")
	}
	return nil
}

func requireUserID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, ok := middleware.UserID(r.Context())
	if !ok {
		writeError(w, nil, model.ErrUnauthenticated)
		return 0, false
	}
	return id, true
}

func (h *handlers) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func parsePositiveInt64(s string) (int64, error) {
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil || n <= 0 {
		return 0, err
	}
	return n, nil
}

func parsePagination(q url.Values) (limit, offset int32, err error) {
	limit = 20
	if s := q.Get("limit"); s != "" {
		n, perr := strconv.ParseInt(s, 10, 64)
		if perr != nil {
			return 0, 0, newAPIError(http.StatusBadRequest, "invalid_request", "invalid limit")
		}
		if n < 1 {
			n = 1
		}
		if n > 100 {
			n = 100
		}
		limit = int32(n)
	}
	if s := q.Get("offset"); s != "" {
		n, perr := strconv.ParseInt(s, 10, 64)
		if perr != nil || n < 0 {
			return 0, 0, newAPIError(http.StatusBadRequest, "invalid_request", "invalid offset")
		}
		offset = int32(n)
	}
	return limit, offset, nil
}
