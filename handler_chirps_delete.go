package main

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerDeleteChirp(w http.ResponseWriter, r *http.Request) {
	// Is user authenticated?
	userID, status, err := cfg.authenticateUser(r)
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
	if chirp.UserID != userID {
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



