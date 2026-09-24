package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/rodyfish/bootdev_chirpy/internal/auth"
	"github.com/rodyfish/bootdev_chirpy/internal/database"
)

type User struct {
	ID uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email string `json:"email"`
	Password string `json:"-"`
	IsChirpyRed bool `json:"is_chirpy_red"`
}

type UserLogin struct {
	ID uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email string `json:"email"`
	IsChirpyRed bool `json:"is_chirpy_red"`
	Token string `json:"token"`
	RefreshToken string `json:"refresh_token"`
}

func (cfg *apiConfig) authenticateUser(r *http.Request) (uuid.UUID, int, error) {
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		return uuid.Nil, http.StatusUnauthorized, errors.New("No token identified: "+err.Error())
	}

	userId, err := auth.ValidateJWT(token, cfg.secret)
	if err != nil {
		return uuid.Nil, http.StatusUnauthorized, errors.New("User could not be identified: "+err.Error())
	}

	return userId, http.StatusOK, nil
}

func (cfg *apiConfig) handlerCreateUser(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Password string `json:"password"`
		Email string `json:"email"`
	}
	type response struct {
		User
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't decode parameters", err)
		return
	}

	hashedPassword, err := auth.HashPassword(params.Password)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error(), err)
		return
	}
	
	dbUser, err := cfg.db.CreateUser(r.Context(), database.CreateUserParams{
		ID: uuid.New(), 
		Email: params.Email, 
		HashedPassword: hashedPassword,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Something went wrong. User couldn't be created.", err)
		return
	}

	respondWithJSON(w, 201, dbUserToUser(dbUser))
}

func dbUserToUser(dbUser database.User) User {
	return User{
		ID: dbUser.ID,
		CreatedAt: dbUser.CreatedAt,
		UpdatedAt: dbUser.UpdatedAt,
		Email: dbUser.Email,
		IsChirpyRed: dbUser.IsChirpyRed,
	}
}