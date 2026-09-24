package main

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/rodyfish/bootdev_chirpy/internal/auth"
)

func (cfg *apiConfig) handlerPolkaWebhooks(w http.ResponseWriter, r* http.Request) {
	// Authentication
	key, err := auth.GetAPIKey(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "", err)
		return
	}
	if key != cfg.polkaKey {
		respondWithError(w, http.StatusUnauthorized, "", errors.New("Apikey doesn't match"))
		return
	}
	
	type parameters struct {
		Event string `json:"event"`
		Data struct {
			UserId uuid.UUID `json:"user_id"`
		} `json:"data"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err = decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't decode parameters", err)
		return
	}

	if params.Event != "user.upgraded" {
		respondWithJSON(w, http.StatusNoContent, "")
		return
	}

	_, err = cfg.db.UpdateToChirpyRed(r.Context(), params.Data.UserId)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "User could not be found", err)
		return
	}

	respondWithJSON(w, http.StatusNoContent, "")
}