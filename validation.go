package main

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"

	"github.com/Priske/chirpy/internal/auth"
	"github.com/Priske/chirpy/internal/database"
	"github.com/google/uuid"
)

func handlerValidateChirp(w http.ResponseWriter, r *http.Request) {
	const (
		maxChirpLength = 140
	)

	var req validateChirpRequest

	err := decodeJSON(r, &req)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Something went wrong")
		return
	}

	if len(req.Body) > maxChirpLength {
		respondWithError(w, http.StatusBadRequest, "Chirp is too long")
		return
	}
	req.Body = replaceBadWord(req.Body)
	respondWithJSON(w, http.StatusOK, cleanedResponse{CleanedBody: req.Body})
}
func (cfg *apiConfig) handlerDeleteChirpByID(w http.ResponseWriter, r *http.Request) {
	chirpIDStr := r.PathValue("chirpID")
	chirpID, err := uuid.Parse(chirpIDStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid chirp ID")
		return
	}

	token, err := auth.GetBearerToken(r.Header) // <-- FIX: r.Header

	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Missing or invalid token")
		return
	}
	tokenUserID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid Token")
		return
	}
	chirp, err := cfg.dbQueries.GetChirpById(r.Context(), chirpID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Chirp id not found")
		return
	}
	// ✅ ownership check (this is what the test wants)
	if chirp.UserID != tokenUserID {
		respondWithError(w, http.StatusForbidden, "You are not the creator of this chirp and cannot delete it")
		return
	}
	if err := cfg.dbQueries.DeleteChirpById(r.Context(), chirpID); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not delete chirp")
		return
	}
	// ✅ 204 should have no response body
	w.WriteHeader(http.StatusNoContent)
}
func (cfg *apiConfig) handlerGetChirpByID(w http.ResponseWriter, r *http.Request) {
	chirpIDStr := r.PathValue("chirpID")

	chirpID, err := uuid.Parse(chirpIDStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid chirp ID")
		return
	}

	dbChirp, err := cfg.dbQueries.GetChirpById(r.Context(), chirpID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Chirp not found")
		return
	}

	respondWithJSON(w, http.StatusOK, dbChirpToAPIChirp(dbChirp))
}

func (cfg *apiConfig) handlerCreateChirp(w http.ResponseWriter, r *http.Request) {
	const maxChirpLength = 140

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Missing or invalid token")
		return
	}
	tokenUserID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid token")
		return
	}

	var req chirpRequest
	if err := decodeJSON(r, &req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if len(req.Body) > maxChirpLength {
		respondWithError(w, http.StatusBadRequest, "Chirp is too long")
		return
	}

	cleanBody := replaceBadWord(req.Body)

	c, err := cfg.dbQueries.CreateChirp(r.Context(), database.CreateChirpParams{
		Body:   cleanBody,
		UserID: tokenUserID,
	})
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Failed to create chirp")
		return
	}

	respondWithJSON(w, http.StatusCreated, apiChirp{
		ID:        c.ID,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
		Body:      c.Body,
		UserId:    c.UserID,
	})
}
func (cfg *apiConfig) handlerFetchAllChirps(w http.ResponseWriter, r *http.Request) {
	authorIDStr := r.URL.Query().Get("author_id")
	sortParam := r.URL.Query().Get("sort")

	// If filtering by author
	if authorIDStr != "" {
		authorID, err := uuid.Parse(authorIDStr)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid author_id")
			return
		}

		dbChirps, err := cfg.dbQueries.GetChirpsByAuthorID(r.Context(), authorID)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Could not fetch chirps")
			return
		}

		sortChirps(dbChirps, sortParam)
		respondWithJSON(w, http.StatusOK, dbChirpsToAPIChirps(dbChirps))
		return
	}

	// Otherwise fetch all
	dbChirps, err := cfg.dbQueries.GetAllChirps(r.Context())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to fetch all chirps")
		return
	}

	sortChirps(dbChirps, sortParam)
	respondWithJSON(w, http.StatusOK, dbChirpsToAPIChirps(dbChirps))
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(payload)
}

func respondWithError(w http.ResponseWriter, code int, msg string) {
	respondWithJSON(w, code, errorResponse{
		Error: msg,
	})
}

func replaceBadWord(msg string) string {
	var bannedWords = map[string]struct{}{
		"kerfuffle": {},
		"sharbert":  {},
		"fornax":    {},
	}
	words := strings.Split(msg, " ")

	for i, word := range words {
		if _, exists := bannedWords[strings.ToLower(word)]; exists {
			words[i] = "****"
		}
	}
	return strings.Join(words, " ")

}
func requireMethod(method string, h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != method {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		h(w, r)
	}
}

func dbChirpToAPIChirp(c database.Chirp) apiChirp {
	return apiChirp{
		ID:        c.ID,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
		Body:      c.Body,
		UserId:    c.UserID,
	}
}
func dbChirpsToAPIChirps(chirps []database.Chirp) []apiChirp {
	result := make([]apiChirp, len(chirps))
	for i, c := range chirps {
		result[i] = dbChirpToAPIChirp(c)
	}
	return result
}

func sortChirps(chirps []database.Chirp, sortParam string) {
	if sortParam == "desc" {
		sort.Slice(chirps, func(i, j int) bool {
			return chirps[i].CreatedAt.After(chirps[j].CreatedAt)
		})
		return
	}

	// default to ascending
	sort.Slice(chirps, func(i, j int) bool {
		return chirps[i].CreatedAt.Before(chirps[j].CreatedAt)
	})
}
