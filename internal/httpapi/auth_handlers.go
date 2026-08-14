package httpapi

import "net/http"

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

// Register registers a new user.
// @Summary Register a new user
// @Tags auth
// @Accept json
// @Produce json
// @Param body body registerRequest true "Registration payload"
// @Success 201 {object} userResponse
// @Failure 400 {object} errorResponse
// @Failure 409 {object} errorResponse
// @Router /register [post]
func (h *handlers) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := decodeBody(w, r, &req, h.maxBodyBytes); err != nil {
		writeError(w, h.logger, err)
		return
	}
	user, err := h.app.Register(r.Context(), req.Email, req.Password, req.Name)
	if err != nil {
		writeError(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusCreated, userResponse{User: user})
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Login authenticates a user and returns a JWT.
// @Summary Login and get a JWT
// @Tags auth
// @Accept json
// @Produce json
// @Param body body loginRequest true "Login payload"
// @Success 200 {object} authResponse
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Router /login [post]
func (h *handlers) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := decodeBody(w, r, &req, h.maxBodyBytes); err != nil {
		writeError(w, h.logger, err)
		return
	}
	token, user, err := h.app.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		writeError(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, authResponse{Token: token, User: user})
}