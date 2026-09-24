package main

import (
	"database/sql"
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
		Email string `json:"email"`
		Password string `json:"password"`
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
	
	dbUser, err := cfg.db.CreateUser(r.Context(), database.CreateUserParams{ID: uuid.New(), Email: params.Email, HashedPassword: hashedPassword})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Something went wrong. User couldn't be created.", err)
		return
	}

	respondWithJSON(w, 201, dbUserToUser(dbUser))
}

func (cfg *apiConfig) handlerUpdateUser(w http.ResponseWriter, r *http.Request){
	type parameters struct {
		Email string `json:"email"`
		Password string `json:"password"`
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find token", err)
		return
	}

	userId, err := auth.ValidateJWT(token, cfg.secret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "User could not be identified", err)
		return
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err = decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't decode parameters", err)
		return
	}

	hashedPassword, err := auth.HashPassword(params.Password)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error(), err)
		return
	}

	user, err := cfg.db.UpdateUserPassword(r.Context(), database.UpdateUserPasswordParams{
		HashedPassword: hashedPassword,
		ID: userId,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error(), err)
		return
	}

	user, err  = cfg.db.UpdateUserEmail(r.Context(), database.UpdateUserEmailParams{
		Email: params.Email,
		ID: user.ID,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error(), err)
		return
	}

	respondWithJSON(w, http.StatusOK, dbUserToUser(user))
}

func (cfg *apiConfig) handlerLogin(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email string `json:"email"`
		Password string `json:"password"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't decode parameters", err)
		return
	}

	dbUser, err := cfg.db.GetUserByEmail(r.Context(), params.Email)
	if err != nil {
		respondWithError(w, 401, "User don't exist", err)
		return
	}
	
	hashCheck, err := auth.CheckPasswordHash(params.Password, dbUser.HashedPassword)
	if !hashCheck {
		respondWithError(w, 401, "Password doesn't match", err)
		return
	}

	token, err := auth.MakeJWT(dbUser.ID, cfg.secret, time.Hour)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error creating token", err)
		return
	}

	refreshToken, err := cfg.db.CreateRefreshToken(r. Context(), database.CreateRefreshTokenParams{
		Token: auth.MakeRefreshToken(),
		ExpiresAt: time.Now().AddDate(0, 0, 60),
		RevokedAt: sql.NullTime{Valid: false},
		UserID: dbUser.ID,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error creating refresh token", err)
		return
	}

	respondWithJSON(w, 200, UserLogin{
		ID: dbUser.ID,
		CreatedAt: dbUser.CreatedAt,
		UpdatedAt: dbUser.UpdatedAt,
		Email: params.Email,
		IsChirpyRed: dbUser.IsChirpyRed,
		Token: token,
		RefreshToken: refreshToken.Token,
	})
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
