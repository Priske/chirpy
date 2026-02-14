package main

import (
	"net/http"
	"time"

	"github.com/Priske/chirpy/internal/auth"
	"github.com/Priske/chirpy/internal/database"
)

func handlerReadiness(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(http.StatusText(http.StatusOK)))
}
func (cfg *apiConfig) handlerRefresh(w http.ResponseWriter, r *http.Request) {
	token, err := auth.GetBearerToken(r.Header) // <-- FIX: r.Header
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Missing or invalid token")
		return
	}

	rt, err := cfg.dbQueries.GetValidRefreshToken(r.Context(), token)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid token")
		return
	}

	accessToken, err := auth.MakeJWT(rt.UserID, cfg.jwtSecret, time.Hour)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create token")
		return
	}

	respondWithJSON(w, http.StatusOK, tokenOnlyResponse{Token: accessToken})
}

func (cfg *apiConfig) handlerRevoke(w http.ResponseWriter, r *http.Request) {
	token, err := auth.GetBearerToken(r.Header) // <-- r.Header
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Missing or invalid token")
		return
	}

	err = cfg.dbQueries.RevokeRefreshToken(r.Context(), token)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't revoke token")
		return
	}

	w.WriteHeader(http.StatusNoContent) // 204
}
func (cfg *apiConfig) handlerUpdateLogin(w http.ResponseWriter, r *http.Request) {

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
	var req loginRequest
	if err := decodeJSON(r, &req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Something went wrong")
		return
	}
	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not hash password")
		return
	}
	user, err := cfg.dbQueries.UpdateUserInfo(r.Context(), database.UpdateUserInfoParams{
		ID:             tokenUserID,
		Email:          req.Email,
		HashedPassword: hashedPassword,
	})
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Could not update the userinfo")
		return
	}

	respondWithJSON(w, http.StatusOK, toAPIUser(user))

}
func (cfg *apiConfig) handlerLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest

	if err := decodeJSON(r, &req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Something went wrong")
		return
	}

	user, err := cfg.dbQueries.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Incorrect email or password")
		return
	}
	expires := time.Hour // default
	if req.ExpiresInSeconds != nil {
		expires = time.Duration(*req.ExpiresInSeconds) * time.Second
	}
	match, err := auth.CheckPasswordHash(req.Password, user.HashedPassword)
	if err != nil || !match {
		respondWithError(w, http.StatusUnauthorized, "Incorrect email or password")
		return
	}
	token, err := auth.MakeJWT(user.ID, cfg.jwtSecret, expires)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	rtoken, err := auth.MakeRefreshToken()
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	// refresh tokens should live longer than access tokens (Boot.dev usually expects ~60 days)
	refreshExpiresAt := time.Now().UTC().Add(60 * 24 * time.Hour)

	_, err = cfg.dbQueries.CreateRefreshToken(r.Context(), database.CreateRefreshTokenParams{
		Token:     rtoken,
		UserID:    user.ID,
		ExpiresAt: refreshExpiresAt,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not save refresh token")
		return
	}

	respondWithJSON(w, http.StatusOK, loginReponseV2ElectricBoogaloo{loginResponse: loginResponse{
		apiUser: toAPIUser(user),
		Token:   token,
	}, Refresh_token: rtoken})
}
