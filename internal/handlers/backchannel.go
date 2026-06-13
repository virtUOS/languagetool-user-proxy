package handlers

import (
	"log"
	"net/http"
)

func (h *UIHandler) BackchannelLogout(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	rawToken := r.FormValue("logout_token")
	if rawToken == "" {
		http.Error(w, "Missing logout_token", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	sub, err := h.OIDCProvider.ValidateLogoutToken(ctx, rawToken)
	if err != nil {
		log.Printf("Back-channel logout: invalid logout token: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	if _, err := h.SessionManager.DeleteSessionsByOIDCSub(ctx, sub); err != nil {
		log.Printf("Back-channel logout: failed to delete sessions for sub %s: %v", sub, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
