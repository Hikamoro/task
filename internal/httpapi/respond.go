package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"task/internal/model"
	"task/internal/service"
)

type errorBody struct {
	Error errorInfo `json:"error"`
}

type errorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type apiError struct {
	status int
	code   string
	msg    string
}

func (e *apiError) Error() string { return e.msg }

func newAPIError(status int, code, msg string) error {
	return &apiError{status: status, code: code, msg: msg}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, logger *slog.Logger, err error) {
	status, code, msg := httpErrorInfo(err)
	if status >= 500 {
		logger.Error("internal error", "error", err)
	}
	writeJSON(w, status, errorBody{Error: errorInfo{Code: code, Message: msg}})
}

func httpErrorInfo(err error) (int, string, string) {
	var ae *apiError
	if errors.As(err, &ae) {
		return ae.status, ae.code, ae.msg
	}
	switch {
	case errors.Is(err, service.ErrInvalidCreds):
		return http.StatusUnauthorized, "invalid_credentials", "invalid email or password"
	case errors.Is(err, service.ErrEmailTaken):
		return http.StatusConflict, "email_taken", "email already registered"
	case errors.Is(err, model.ErrUnauthenticated):
		return http.StatusUnauthorized, "unauthorized", "authentication required"
	case errors.Is(err, model.ErrInvalidInput):
		return http.StatusBadRequest, "invalid_request", "invalid input"
	case errors.Is(err, model.ErrForbidden):
		return http.StatusForbidden, "forbidden", "insufficient permissions"
	case errors.Is(err, model.ErrNotFound):
		return http.StatusNotFound, "not_found", "resource not found"
	case errors.Is(err, model.ErrConflict):
		return http.StatusConflict, "conflict", "resource was modified concurrently; retry with the latest version"
	case errors.Is(err, model.ErrDuplicate):
		return http.StatusConflict, "conflict", "resource already exists"
	default:
		return http.StatusInternalServerError, "internal", "internal server error"
	}
}
