package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"shtnr/db"

	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) *db.Queries {
	sqlDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}

	_, err = sqlDB.Exec(ddl)
	if err != nil {
		t.Fatal(err)
	}

	return db.New(sqlDB)
}

func TestShortenHandler(t *testing.T) {
	queries := setupTestDB(t)
	server := &Server{db: queries}

	reqBody := ShortenRequest{URL: "https://example.com"}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/shorten", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.shortenHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp ShortenResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}

	if resp.ShortURL == "" {
		t.Error("Expected short URL in response")
	}
}

func TestRedirectHandler(t *testing.T) {
	queries := setupTestDB(t)
	server := &Server{db: queries}

	shortCode := "test123"
	originalURL := "https://example.com"

	_, err := queries.CreateURL(context.Background(), db.CreateURLParams{
		ShortCode:   shortCode,
		OriginalUrl: originalURL,
	})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/"+shortCode, nil)
	w := httptest.NewRecorder()

	server.redirectHandler(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("Expected status 302, got %d", w.Code)
	}

	location := w.Header().Get("Location")
	if location != originalURL {
		t.Errorf("Expected redirect to %s, got %s", originalURL, location)
	}

	url, err := queries.GetURL(context.Background(), shortCode)
	if err != nil {
		t.Fatal(err)
	}

	if url.AccessCount.Int64 != 1 {
		t.Errorf("Expected access count 1, got %d", url.AccessCount.Int64)
	}
}

func TestRedirectHandlerNotFound(t *testing.T) {
	queries := setupTestDB(t)
	server := &Server{db: queries}

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	w := httptest.NewRecorder()

	server.redirectHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestShortenHandlerInvalidMethod(t *testing.T) {
	queries := setupTestDB(t)
	server := &Server{db: queries}

	req := httptest.NewRequest(http.MethodGet, "/shorten", nil)
	w := httptest.NewRecorder()

	server.shortenHandler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

func TestShortenHandlerInvalidJSON(t *testing.T) {
	queries := setupTestDB(t)
	server := &Server{db: queries}

	req := httptest.NewRequest(http.MethodPost, "/shorten", bytes.NewReader([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.shortenHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestShortenHandlerEmptyURL(t *testing.T) {
	queries := setupTestDB(t)
	server := &Server{db: queries}

	reqBody := ShortenRequest{URL: ""}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/shorten", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.shortenHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}
