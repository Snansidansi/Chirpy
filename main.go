package main

import (
	"compress/gzip"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"sync/atomic"

	"github.com/Snansidansi/Chirpy/internal/database"
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

	dbQueries := database.New(db)
	_ = dbQueries

	apiCfg := apiConfig{
		fileserverHits: atomic.Int32{},
	}

	serveMux := http.NewServeMux()
	serveMux.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app/", http.FileServer(http.Dir(filePathRoot)))))
	serveMux.HandleFunc("GET /admin/healthz", handleHealth)
	serveMux.HandleFunc("GET /admin/metrics", apiCfg.handleGetMetrics)
	serveMux.HandleFunc("POST /admin/reset", apiCfg.handleResetMetrics)
	serveMux.Handle("POST /api/validate_chirp", apiCfg.middlewareMetricsInc(http.HandlerFunc(handleValidateChrip)))

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

func (cfg *apiConfig) handleResetMetrics(w http.ResponseWriter, _ *http.Request) {
	cfg.fileserverHits.Swap(0)

	w.WriteHeader(200)
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte("Successfuly reset request counter."))
}

func handleValidateChrip(w http.ResponseWriter, r *http.Request) {
	type chirp struct {
		Body string `json:"body"`
	}

	w.Header().Add("Content-Type", "application/json")

	decoder := json.NewDecoder(r.Body)
	rChirp := chirp{}
	if err := decoder.Decode(&rChirp); err != nil {
		w.WriteHeader(500)
		w.Write([]byte(`{"error": "Something went wrong"}`))
		return
	}

	if len(rChirp.Body) > 140 {
		w.WriteHeader(400)
		w.Write([]byte(`{"error": "Chirp is too long"}`))
		return
	}

	for _, word := range []string{"kerfuffle", "sharbert", "fornax"} {
		re := regexp.MustCompile(fmt.Sprintf(`(?i)%s`, word))
		rChirp.Body = re.ReplaceAllString(rChirp.Body, "****")
	}

	w.Header().Add("Content-Encoding", "gzip")
	w.WriteHeader(200)

	gzw := gzip.NewWriter(w)
	defer gzw.Close()
	_, err := gzw.Write(fmt.Appendf(nil, `{"cleaned_body": "%s"}`, rChirp.Body))
	if err != nil {
		fmt.Println(err)
	}
}
