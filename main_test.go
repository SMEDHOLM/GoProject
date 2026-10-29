package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type memoryArticleStore struct {
	articles []Article
}

func (s *memoryArticleStore) List(context.Context) ([]Article, error) {
	return s.articles, nil
}

func (s *memoryArticleStore) Get(_ context.Context, id string) (Article, error) {
	for _, article := range s.articles {
		if article.ID == id {
			return article, nil
		}
	}
	return Article{}, errArticleNotFound
}

func (s *memoryArticleStore) Create(_ context.Context, article Article) error {
	article.ID = "1"
	s.articles = append(s.articles, article)
	return nil
}

func (s *memoryArticleStore) Close() error {
	return nil
}

func TestArticleRoutes(t *testing.T) {
	store := &memoryArticleStore{
		articles: []Article{{
			ID:       "1",
			Title:    "Test article",
			Anons:    "A short summary",
			FullText: "Full article text",
		}},
	}
	handler := newRouter(store)

	tests := []struct {
		name         string
		method       string
		path         string
		body         string
		wantStatus   int
		wantBodyPart string
	}{
		{name: "homepage", method: http.MethodGet, path: "/", wantStatus: http.StatusOK, wantBodyPart: "Test article"},
		{name: "create page", method: http.MethodGet, path: "/create", wantStatus: http.StatusOK, wantBodyPart: "Enter Title"},
		{name: "post page", method: http.MethodGet, path: "/post/1", wantStatus: http.StatusOK, wantBodyPart: "Full article text"},
		{name: "missing post", method: http.MethodGet, path: "/post/2", wantStatus: http.StatusNotFound},
		{name: "create article", method: http.MethodPost, path: "/save_article", body: "title=Another&anons=Summary&full_text=Body", wantStatus: http.StatusSeeOther},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			if tt.method == http.MethodPost {
				request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			}
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			if response.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", response.Code, tt.wantStatus, response.Body.String())
			}
			if tt.wantBodyPart != "" && !strings.Contains(response.Body.String(), tt.wantBodyPart) {
				t.Fatalf("response does not contain %q: %s", tt.wantBodyPart, response.Body.String())
			}
		})
	}

	if len(store.articles) != 2 {
		t.Fatalf("stored articles = %d, want 2", len(store.articles))
	}
}
