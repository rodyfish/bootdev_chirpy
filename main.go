package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync/atomic"
	"time"

	"github.com/joho/godotenv"

	_ "github.com/lib/pq"
	"github.com/rodyfish/bootdev_chirpy/internal/auth"
	"github.com/rodyfish/bootdev_chirpy/internal/database"
)



type apiConfig struct {
	fileserverHits atomic.Int32
	db *database.Queries
	platform string
	secret string
	polkaKey string
}

func main() {
	const port = "8080"
	const filepathRoot = "."

	godotenv.Load()

	dbURL := getEnvOrFatal("DB_URL")
	platform := getEnvOrFatal("PLATFORM")
	jwtSecret := getEnvOrFatal("SECRET")
	polkaKey := getEnvOrFatal("POLKA_KEY")
	
	dbConn, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Error opening database: %s", err)
	}

	dbQueries := database.New(dbConn)
	
	apiCfg := &apiConfig{
		db: dbQueries, 
		platform: platform,
		secret: jwtSecret,
		polkaKey: polkaKey,
	}

	// ServeMux is the Router/traffic cop for the server. 
	// Its job is to look at the URL path and HTTP method of an incoming 
	// request and route it to the specific code that knows how to handle it.


	// 1. Create the router (mux) to direct traffic
	mux := http.NewServeMux()
	fsHandler := apiCfg.middlewareMetricInc(http.StripPrefix("/app", http.FileServer(http.Dir(filepathRoot))))
	// Strip "/app/" (with trailing slash) so file server receives relative paths
	mux.Handle("/app/", fsHandler)

	// Adding handler. A Handler is a rule that looks for a pattern, which then serve files from .Dir.
	mux.HandleFunc("GET /api/chirps", apiCfg.handlerGetAllChirps)
	mux.HandleFunc("GET /api/chirps/{chirpID}", apiCfg.handlerGetChirp)
	mux.HandleFunc("POST /api/chirps", apiCfg.handlerCreateChirp)
	mux.HandleFunc("DELETE /api/chirps/{chirpID}", apiCfg.handlerDeleteChirp)
	mux.HandleFunc("GET /api/healthz", handlerReadiness)
	mux.HandleFunc("POST /api/login", apiCfg.handlerLogin)
	mux.HandleFunc("POST /api/polka/webhooks", apiCfg.handlerPolkaWebhooks)
	mux.HandleFunc("POST /api/refresh", apiCfg.handlerRefresh)
	mux.HandleFunc("POST /api/revoke", apiCfg.handlerRevoke)
	mux.HandleFunc("POST /api/users", apiCfg.handlerCreateUser)
	mux.HandleFunc("PUT /api/users", apiCfg.handlerUpdateUser)
	

	mux.HandleFunc("GET /admin/metrics", apiCfg.handlerShowCount)
	mux.HandleFunc("POST /admin/reset", apiCfg.handlerReset)


	// 2. Create the server and hand it the router
	server := &http.Server{
		Handler: mux,
		Addr: ":"+port,
	}
	
	err = server.ListenAndServe()
	if err != nil {
		log.Fatalln("Server failed starting..")
	}
}

func getEnvOrFatal(key string) string {
	val := os.Getenv(key)
	if val == "" {
		log.Fatalf("%s environment variable is not set", key)
	}
	return val
}

func handlerReadiness(writer http.ResponseWriter, req *http.Request) {
	writer.Header().Add("Content-Type", "text/plain; charset=utf-8")
	writer.WriteHeader(http.StatusOK)
	writer.Write([]byte(http.StatusText(http.StatusOK)))
}

func (cfg *apiConfig) handlerShowCount(w http.ResponseWriter, req* http.Request) {
	html := fmt.Sprintf(
	`<html>
  		<body>
			<h1>Welcome, Chirpy Admin</h1>
			<p>Chirpy has been visited %d times!</p>
		</body>
	</html>`, cfg.fileserverHits.Load())

	w.Header().Add("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	
	w.Write([]byte(html))
}

func (cfg *apiConfig) handlerRefresh(w http.ResponseWriter, r* http.Request) {
	type responseVals struct {
		Token string `json:"token"`
	}

	refreshTokenString, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "No Bearer token recieved", err)
		return
	}

	rt, err := cfg.db.GetRefreshToken(r.Context(), refreshTokenString)
	if errors.Is(err, sql.ErrNoRows) {
		respondWithError(w, http.StatusUnauthorized, "Refresh token is invalid", err)
		return
	} else if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error(), err)
		return
	}
	
	test1 := rt.ExpiresAt.Before(time.Now()) 
	test2 := rt.RevokedAt.Valid
	test3 := rt.RevokedAt.Time.Before(time.Now())
	if test1 || (test2 && test3) {
		respondWithError(w, http.StatusUnauthorized, "Token is expired..", err)
		return
	}

	newToken, err := auth.MakeJWT(rt.UserID, cfg.secret, time.Hour)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't renew token", err)
		return
	}

	respondWithJSON(w, http.StatusOK, responseVals{Token: newToken})
}

func (cfg *apiConfig) handlerRevoke(w http.ResponseWriter, r* http.Request) {
	refreshTokenString, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "No Bearer token recieved", err)
		return
	}

	rt, err := cfg.db.GetRefreshToken(r.Context(), refreshTokenString)
	if errors.Is(err, sql.ErrNoRows) {
		respondWithError(w, http.StatusUnauthorized, "Refresh token is invalid", err)
		return
	} else if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error(), err)
		return
	}
		
	rt, err = cfg.db.RevokeRefreshToken(r.Context(), refreshTokenString)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error(), err)
		return
	}
		
	fmt.Println("Handler revoke token: ", refreshTokenString, rt.Token, rt.ExpiresAt, rt.UpdatedAt, rt.RevokedAt.Time)
	fmt.Println("")
	respondWithJSON(w, http.StatusNoContent, "")

}

func (cfg *apiConfig) middlewareMetricInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}



