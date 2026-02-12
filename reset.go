package main

import (
	"net/http"
)

func (cfg *apiConfig) handlerReset(w http.ResponseWriter, r *http.Request) {
	if cfg.platform != "dev" {
		respondWithError(w, http.StatusForbidden, "Forbidden")
		return
	}

	// Delete users first
	if err := cfg.dbQueries.ResetUsers(r.Context()); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not delete users")
		return
	}

	// Reset metrics (if still required)
	cfg.fileserverHits.Store(0)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Hits reset to 0"))
}
