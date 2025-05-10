package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"
)

func main() {
	const (
		filePathRoot = "."
		port         = "8080"
	)

	apiCfg := apiConfig{
		fileserverHits: atomic.Int32{},
	}

	serveMux := http.NewServeMux()
	serveMux.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app/", http.FileServer(http.Dir(filePathRoot)))))
	serveMux.HandleFunc("GET /api/healthz", handleHealth)
	serveMux.HandleFunc("GET /api/metrics", apiCfg.handleGetMetrics)
	serveMux.HandleFunc("POST /api/reset", apiCfg.handleResetMetrics)
	serveMux.HandleFunc("POST /api/validate_chirp", handleValidateChrip)

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
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")

	message := fmt.Sprintf("Hits: %v", cfg.fileserverHits.Load())
	w.Write([]byte(message))
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

	w.WriteHeader(200)
	w.Write([]byte(`{"valid": true}`))
}
