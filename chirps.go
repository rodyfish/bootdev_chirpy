package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rodyfish/bootdev_chirpy/internal/auth"
	"github.com/rodyfish/bootdev_chirpy/internal/database"
)

type Chirp struct {
	ID uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body string `json:"body"`
	UserId uuid.UUID `json:"user_id"`
}

func (cfg *apiConfig) handlerCreateChirp(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Body string `json:"body"`
		UserId uuid.UUID `json:"user_id"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't decode parameters from chirp", err)
		return
	} 

	// Checks if user is authenticated
	bearerToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error(), err)
		return
	}

	validatedUserId, err := auth.ValidateJWT(bearerToken, cfg.secret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, err.Error(), err)
		return
	}
	params.UserId = validatedUserId

	cleanedBody, err := validateChirp(params.Body)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error(), err)
		return
	}
	
	newChirp, err := cfg.db.CreateChirp(r.Context(), database.CreateChirpParams{
		ID: uuid.New(),
		Body: cleanedBody,
		UserID: params.UserId,
	})

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Something went wrong. Couldn't create chirp", err)
		return
	}

	respondWithJSON(w, 201, dbChirpToChirp(newChirp))
}

func validateChirp(body string) (string, error) {
	const maxChirpLength = 140
	if len(body) > maxChirpLength {
		return "", errors.New("Chirp it too long. Max length is 140 chars")
	}

	cleanedBody, _ := cleanProfaneWords(body)
	return cleanedBody, nil
}

func cleanProfaneWords(body string) (cleanedBody string, foundWords []string) {
	profaneWords := map[string]struct{}{
		"kerfuffle": {},
		"sharbert": {},
		"fornax": {},
	}
	words := strings.Split(body, " ")
	cleanedWords := []string{}
	foundProfaneWords := []string{}

	for _, word := range words {
		if _, ok := profaneWords[strings.ToLower(word)]; ok {
			foundProfaneWords = append(foundProfaneWords, word)
			word = "****"
		}
		cleanedWords = append(cleanedWords, word)
	}

	return strings.Join(cleanedWords, " "), foundProfaneWords
}

func (cfg *apiConfig) handlerDeleteChirp(w http.ResponseWriter, r *http.Request) {
	// Is user authenticated?
	userId, status, err := cfg.authenticateUser(r)
	if err != nil {
		respondWithError(w, status, err.Error(), err)
		return
	}
	
	// Is input data valid?
	rawChirpId := r.PathValue("chirpID")
	fmt.Println("Delete chirp", rawChirpId, r.URL.RawPath)
	chirpId, err := uuid.Parse(rawChirpId)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error(), err)
		return
	}

	// Does chirp exist?
	chirp, err := cfg.db.GetChirp(r.Context(), chirpId)
	if err != nil {
		respondWithError(w, http.StatusNotFound, err.Error(), err)
		return
	}

	// Does chirp belong to user?
	if chirp.UserID != userId {
		respondWithError(w, http.StatusForbidden, "You are not allowed to delete this chirp, since you're not the owner", nil)
		return
	}

	err = cfg.db.DeleteChirp(r.Context(), chirpId)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error(), err)
		return
	}

	respondWithJSON(w, http.StatusNoContent, "")
}



func dbChirpToChirp (dbChirp database.Chirp) Chirp {
	return Chirp{
		ID: dbChirp.ID,
		CreatedAt: dbChirp.CreatedAt,
		UpdatedAt: dbChirp.UpdatedAt,
		Body: dbChirp.Body,
		UserId: dbChirp.UserID,
	}
}