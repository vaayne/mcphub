package skills

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSearchSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("q") != "react" {
			t.Errorf("expected query=react, got %s", r.URL.Query().Get("q"))
		}
		_ = json.NewEncoder(w).Encode(searchResponse{
			Count: 1,
			Skills: []SearchResult{
				{ID: "1", SkillID: "react-skill", Name: "React Skill", Installs: 42, Source: "owner/repo"},
			},
		})
	}))
	defer srv.Close()

	results, err := searchWithURL(context.Background(), srv.URL, "react", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].SkillID != "react-skill" {
		t.Errorf("expected skillId=react-skill, got %s", results[0].SkillID)
	}
}

func TestSearchNoResults(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(searchResponse{Count: 0, Skills: []SearchResult{}})
	}))
	defer srv.Close()

	results, err := searchWithURL(context.Background(), srv.URL, "nonexistent", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

func TestSearchDefaultLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("limit") != "10" {
			t.Errorf("expected default limit=10, got %s", r.URL.Query().Get("limit"))
		}
		_ = json.NewEncoder(w).Encode(searchResponse{Count: 0, Skills: []SearchResult{}})
	}))
	defer srv.Close()

	_, err := searchWithURL(context.Background(), srv.URL, "test", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSearchCustomLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("limit") != "5" {
			t.Errorf("expected limit=5, got %s", r.URL.Query().Get("limit"))
		}
		_ = json.NewEncoder(w).Encode(searchResponse{Count: 0, Skills: []SearchResult{}})
	}))
	defer srv.Close()

	_, err := searchWithURL(context.Background(), srv.URL, "test", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSearchAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	_, err := searchWithURL(context.Background(), srv.URL, "test", 10)
	if err == nil {
		t.Fatal("expected error for 500 status")
	}
}

func TestSearchInvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}))
	defer srv.Close()

	_, err := searchWithURL(context.Background(), srv.URL, "test", 10)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}
