package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"sync/atomic"
	"time"

	"github.com/Snansidansi/Chirpy/internal/database"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	const (
		filePathRoot = "."
		port         = "8080"
	)

	godotenv.Load()
	dbURL := os.Getenv("DB_URL")

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal(err)
	}

	apiCfg := apiConfig{
		fileserverHits: atomic.Int32{},
		db:             database.New(db),
		platform:       os.Getenv("PLATFORM"),
	}

	serveMux := http.NewServeMux()
	serveMux.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app/", http.FileServer(http.Dir(filePathRoot)))))
	serveMux.HandleFunc("GET /admin/healthz", handleHealth)
	serveMux.HandleFunc("GET /admin/metrics", apiCfg.handleGetMetrics)
	serveMux.HandleFunc("POST /admin/reset", apiCfg.handlerReset)
	serveMux.HandleFunc("POST /api/chirps", apiCfg.handlerCreateChirp)
	serveMux.HandleFunc("POST /api/users", apiCfg.handlerCreateUser)
	serveMux.HandleFunc("DELETE /api/users", apiCfg.handlerDeleteAllUsers)

	httpServer := http.Server{
		Handler: serveMux,
		Addr:    ":" + port,
	}

	httpServer.ListenAndServe()
}

func handleHealth(responseWriter http.ResponseWriter, _ *http.Request) {
	responseWriter.WriteHeader(200)
	responseWriter.Header().Add("Content-Type", "text/plain; charset=utf-8")
	responseWriter.Write([]byte(http.StatusText(http.StatusOK)))
}

type apiConfig struct {
	fileserverHits atomic.Int32
	db             *database.Queries
	platform       string
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	middlewareHandler := func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(middlewareHandler)
}

func (cfg *apiConfig) handleGetMetrics(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	w.Header().Add("Content-Type", "text/html")

	message := `<html>
				  <body>
					<h1>Welcome, Chirpy Admin</h1>
					<p>Chirpy has been visited %d times!</p>
				  </body>
				</html>`

	w.Write(fmt.Appendf(nil, message, cfg.fileserverHits.Load()))
}

func (cfg *apiConfig) handlerReset(w http.ResponseWriter, _ *http.Request) {
	if cfg.platform != "dev" {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	cfg.fileserverHits.Swap(0)

	err := cfg.db.DeleteAllUsers(context.Background())
	if err != nil {
		respondWithError(w, 500, err)
		return
	}

	w.WriteHeader(200)
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte("Successfuly reset request counter."))
}

func (cfg *apiConfig) handlerCreateChirp(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Body   string    `json:"body"`
		UserID uuid.UUID `json:"user_id"`
	}

	decoder := json.NewDecoder(r.Body)
	var chirp parameters
	if err := decoder.Decode(&chirp); err != nil {
		respondWithError(w, http.StatusBadRequest, err)
		return
	}

	body, err := validateChrip(chirp.Body)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err)
		return
	}
	chirp.Body = body

	createdChrip, err := cfg.db.CreateChirp(context.Background(), database.CreateChirpParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Body:      chirp.Body,
		UserID:    chirp.UserID,
	})
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err)
		return
	}

	respondJson(w, http.StatusCreated, Chirp{
		Id:         createdChrip.ID,
		Created_at: createdChrip.CreatedAt,
		Updated_at: createdChrip.CreatedAt,
		Body:       createdChrip.Body,
		User_id:    createdChrip.UserID,
	})
}

type Chirp struct {
	Id         uuid.UUID `json:"id"`
	Created_at time.Time `json:"created_at"`
	Updated_at time.Time `json:"updated_at"`
	Body       string    `json:"body"`
	User_id    uuid.UUID `json:"user_id"`
}

func validateChrip(body string) (string, error) {
	if len(body) > 140 {
		return "", fmt.Errorf("Chirp is too long")
	}

	for _, word := range []string{"kerfuffle", "sharbert", "fornax"} {
		re := regexp.MustCompile(fmt.Sprintf(`(?i)%s`, word))
		body = re.ReplaceAllString(body, "****")
	}

	return body, nil
}

func (cfg *apiConfig) handlerCreateUser(w http.ResponseWriter, r *http.Request) {
	type userParameters struct {
		Email string `json:"email"`
	}

	var userParams userParameters
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&userParams); err != nil {
		respondWithError(w, 400, err)
		return
	}

	createdUser, err := cfg.db.CreateUser(context.Background(), database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Email:     userParams.Email,
	})
	tempUser := struct {
		Id         uuid.UUID `json:"id"`
		Created_at time.Time `json:"created_at"`
		Updated_at time.Time `json:"updated_at"`
		Email      string    `json:"email"`
	}{
		Id:         createdUser.ID,
		Created_at: createdUser.CreatedAt,
		Updated_at: createdUser.UpdatedAt,
		Email:      createdUser.Email,
	}
	if err != nil {
		respondWithError(w, 400, err)
		return
	}

	respondJson(w, 201, tempUser)
}

func respondWithError(w http.ResponseWriter, httpCode int, err error) {
	type errorResponse struct {
		Message string `json:"message"`
	}
	errResp := errorResponse{
		Message: fmt.Sprint(err),
	}

	respondJson(w, httpCode, errResp)
}

func respondJson(w http.ResponseWriter, httpCode int, body any) {
	data, err := json.Marshal(body)
	if err != nil {
		w.WriteHeader(500)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(httpCode)
	w.Write(data)
}

func (cfg *apiConfig) handlerDeleteAllUsers(w http.ResponseWriter, r *http.Request) {
	err := cfg.db.DeleteAllUsers(context.Background())
	if err != nil {
		respondWithError(w, 500, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
