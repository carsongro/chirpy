package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/carsongro/chirpy/internal/database"
)

func main() {
	const filePathRoot = "."
	const port = "8080"

	apiCfg := apiConfig{
		fileserverHits: 0,
	}

	dbg := flag.Bool("debug", false, "Enable debug mode")
	flag.Parse()

	db, err := database.NewDB("database.json", *dbg)
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.Handle("/app/*", apiCfg.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir(filePathRoot)))))

	mux.HandleFunc("GET /api/healthz", redinessHandler)
	mux.HandleFunc("GET /admin/metrics", apiCfg.metricsHandler)
	mux.HandleFunc("/reset", apiCfg.resetHandler)

	mux.HandleFunc("POST /api/chirps", db.PostChirpHandler)
	mux.HandleFunc("GET /api/chirps", db.GetChirpsHandler)
	mux.HandleFunc("GET /api/chirps/{chirpID}", db.GetChirpHandler)

	mux.HandleFunc("POST /api/users", db.PostUserHandler)
	mux.HandleFunc("POST /api/login", db.PostLoginHandler)

	corsMux := middlewareCors(mux)

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: corsMux,
	}

	log.Printf("Serving on port: %s\n", port)
	log.Fatal(srv.ListenAndServe())
}

func redinessHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(http.StatusText(http.StatusOK)))
}

func (a *apiConfig) metricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "<html><body><h1>Welcome, Chirpy Admin</h1><p>Chirpy has been visited %d times!</p></body></html>", a.fileserverHits)
}

func (a *apiConfig) resetHandler(w http.ResponseWriter, r *http.Request) {
	a.fileserverHits = 0
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(http.StatusText(http.StatusOK)))
}
