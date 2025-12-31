package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"shtnr/db"

	docs "shtnr/docs"

	httpSwagger "github.com/swaggo/http-swagger/v2"
	_ "modernc.org/sqlite"
)

//go:embed sql/schema.sql
var ddl string

type Server struct {
	db *db.Queries
}

type ShortenRequest struct {
	URL string `json:"url" example:"https://www.example.com/very/long/url" binding:"required"`
}

type ShortenResponse struct {
	ShortURL string `json:"short_url" example:"http://localhost:8080/abc123"`
}

func generateShortCode() string {
	bytes := make([]byte, 6)
	rand.Read(bytes)
	return base64.URLEncoding.EncodeToString(bytes)[:8]
}

// shortenHandler godoc
//
//	@Summary		Shorten a URL
//	@Description	Creates a short code for the provided URL
//	@Tags			urls
//	@Accept			json
//	@Produce		json
//	@Param			request	body		ShortenRequest	true	"URL to shorten"
//	@Success		200		{object}	ShortenResponse
//	@Failure		400		{string}	string	"Bad Request"
//	@Failure		405		{string}	string	"Method Not Allowed"
//	@Failure		500		{string}	string	"Internal Server Error"
//	@Router			/shorten [post]
func (s *Server) shortenHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ShortenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.URL == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}

	shortCode := generateShortCode()

	_, err := s.db.CreateURL(context.Background(), db.CreateURLParams{
		ShortCode:   shortCode,
		OriginalUrl: req.URL,
	})
	if err != nil {
		log.Printf("Error creating URL: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	resp := ShortenResponse{
		ShortURL: "http://" + r.Host + "/" + shortCode,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) redirectHandler(w http.ResponseWriter, r *http.Request) {
	shortCode := strings.TrimPrefix(r.URL.Path, "/")
	if shortCode == "" {
		http.NotFound(w, r)
		return
	}

	url, err := s.db.GetURL(context.Background(), shortCode)
	if err != nil {
		if err == sql.ErrNoRows {
			http.NotFound(w, r)
			return
		}
		log.Printf("Error getting URL: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	s.db.UpdateAccessCount(context.Background(), shortCode)

	http.Redirect(w, r, url.OriginalUrl, http.StatusFound)
}

// @title URL Shortener API
// @version 1.0
// @description A simple URL shortening service that creates short codes for long URLs.
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

func main() {
	sqlDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		log.Fatal(err)
	}
	defer sqlDB.Close()

	if err := sqlDB.Ping(); err != nil {
		log.Fatal(err)
	}

	_, err = sqlDB.Exec(ddl)
	if err != nil {
		log.Fatal(err)
	}

	queries := db.New(sqlDB)

	server := &Server{
		db: queries,
	}

	http.HandleFunc("/shorten", server.shortenHandler)
	http.Handle("/swagger/", httpSwagger.Handler(
		httpSwagger.URL("/docs/doc.json"),
	))

	http.HandleFunc("/docs/doc.json", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(docs.SwaggerInfo.ReadDoc()))
	})

	http.HandleFunc("/", server.redirectHandler)

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
