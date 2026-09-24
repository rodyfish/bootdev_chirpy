package main

import (
	"net/http"
	"sort"

	"github.com/google/uuid"
	"github.com/rodyfish/bootdev_chirpy/internal/database"
)

func getAuthorIDFromRequest(r *http.Request) (uuid.UUID, error) {
	authorIDString := r.URL.Query().Get("author_id")
	if authorIDString == "" {
		return uuid.Nil, nil
	}

	authorID, err := uuid.Parse(authorIDString)
	if err != nil {
		return uuid.Nil, err
	}
	
	return authorID, nil
}

func (cfg *apiConfig) handlerGetAllChirps(w http.ResponseWriter, r *http.Request) {
	authorID, err := getAuthorIDFromRequest(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid author ID", err)
		return
	}

	dbChirps := []database.Chirp{}
	if authorID != uuid.Nil {
		dbChirps, err = cfg.db.GetAllChirpsFromUser(r.Context(), authorID)
	} else {
		dbChirps, err = cfg.db.GetAllChirps(r.Context())
	}

	if r.URL.Query().Get("sort") == "desc" {
		sort.Slice(dbChirps, func(i, j int) bool {return dbChirps[i].CreatedAt.After(dbChirps[j].CreatedAt)})
	} else {
		sort.Slice(dbChirps, func(i, j int) bool {return dbChirps[i].CreatedAt.Before(dbChirps[j].CreatedAt)})
	}

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't retrieve chirps", err)
		return
	}

	chirps := []Chirp{}

	for _, dbChirp := range dbChirps {
		chirps = append(chirps, dbChirpToChirp(dbChirp))
	}

	respondWithJSON(w, 200, chirps)
}

func (cfg *apiConfig) handlerGetChirp(w http.ResponseWriter, r *http.Request) {
	
	chirpIDString := r.PathValue("chirpID")
	chirpID, err := uuid.Parse(chirpIDString)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error(), err)
		return
	}
	
	dbChirp, err := cfg.db.GetChirp(r.Context(), chirpID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Couldn't get chirp", err)
		return
	}

	respondWithJSON(w, 200, dbChirpToChirp(dbChirp))
}