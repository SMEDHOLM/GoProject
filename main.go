package main

import (
	"context"
	"embed"
	"errors"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

//go:embed templates/*.html
var templateFiles embed.FS

type Article struct {
	ID       string
	Title    string
	Anons    string
	FullText string
}

type articleStore interface {
	List(context.Context) ([]Article, error)
	Get(context.Context, string) (Article, error)
	Create(context.Context, Article) error
	Close() error
}

func renderTemplate(w http.ResponseWriter, filename, name string, data any) {
	t, err := template.ParseFS(templateFiles, filename, "templates/head.html", "templates/footer.html")
	if err != nil {
		log.Printf("parse %s template: %v", name, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if err := t.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("render %s template: %v", name, err)
	}
}

func newRouter(articles articleStore) http.Handler {
	rtr := mux.NewRouter()
	rtr.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		posts, err := articles.List(r.Context())
		if err != nil {
			log.Printf("list articles: %v", err)
			http.Error(w, "Unable to load articles", http.StatusInternalServerError)
			return
		}
		renderTemplate(w, "templates/index.html", "index", posts)
	}).Methods(http.MethodGet)
	rtr.HandleFunc("/create", func(w http.ResponseWriter, r *http.Request) {
		renderTemplate(w, "templates/create.html", "create", nil)
	}).Methods(http.MethodGet)
	rtr.HandleFunc("/save_article", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid form data", http.StatusBadRequest)
			return
		}

		article := Article{
			Title:    strings.TrimSpace(r.FormValue("title")),
			Anons:    strings.TrimSpace(r.FormValue("anons")),
			FullText: strings.TrimSpace(r.FormValue("full_text")),
		}
		if article.Title == "" || article.Anons == "" || article.FullText == "" {
			http.Error(w, "All article fields are required", http.StatusBadRequest)
			return
		}
		if err := articles.Create(r.Context(), article); err != nil {
			log.Printf("save article: %v", err)
			http.Error(w, "Unable to save article", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}).Methods(http.MethodPost)
	rtr.HandleFunc("/post/{id:[a-fA-F0-9]+}", func(w http.ResponseWriter, r *http.Request) {
		article, err := articles.Get(r.Context(), mux.Vars(r)["id"])
		if errors.Is(err, errArticleNotFound) {
			http.NotFound(w, r)
			return
		}
		if err != nil {
			log.Printf("get article: %v", err)
			http.Error(w, "Unable to load article", http.StatusInternalServerError)
			return
		}
		renderTemplate(w, "templates/post.html", "post", article)
	}).Methods(http.MethodGet)
	return rtr
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	articles, err := openArticleStore(ctx)
	cancel()
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := articles.Close(); err != nil {
			log.Printf("close article database: %v", err)
		}
	}()

	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = "8081"
	}
	addr := ":" + port
	log.Printf("serving on http://localhost:%s", port)
	if err := http.ListenAndServe(addr, newRouter(articles)); err != nil {
		log.Printf("HTTP server stopped: %v", err)
	}
}
