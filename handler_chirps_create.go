package main

import (
	"encoding/json"
	"errors"
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
	}

	// Checks if user is authenticated
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, err.Error(), err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.secret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err = decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't decode parameters", err)
		return
	} 

	cleanedBody, err := validateChirp(params.Body)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error(), err)
		return
	}
	
	chirp, err := cfg.db.CreateChirp(r.Context(), database.CreateChirpParams{
		ID: uuid.New(),
		Body: cleanedBody,
		UserID: userID,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Something went wrong. Couldn't create chirp", err)
		return
	}

	respondWithJSON(w, 201, dbChirpToChirp(chirp))
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