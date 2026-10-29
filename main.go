package main

import (
	"database/sql"
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
)

//go:embed templates/*.html
var templateFiles embed.FS
type Article struct {
	Id uint16
	Title, Anons, FullText string
}

func create(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFS(templateFiles, "templates/create.html", "templates/head.html", "templates/footer.html")
	if err != nil {
		log.Printf("parse create template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if err := t.ExecuteTemplate(w, "create", nil); err != nil {
		log.Printf("render create template: %v", err)
	}
}

func index(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFS(templateFiles, "templates/index.html", "templates/head.html", "templates/footer.html")
	if err != nil {
		log.Printf("parse index template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	db, err := sql.Open("mysql", "root:root@tcp(127.0.0.1:3306)/golang")
		if err != nil {
			panic(err)
		}
		defer db.Close()

	res, err:= db.Query("SELECT * FROM articles")
	if err != nil {
		panic(err)
	}
	var posts = []Article{}
	for res.Next(){
		var article Article
		err = res.Scan(&article.Id, &article.Title, &article.Anons, &article.FullText )
		if err != nil {
		panic(err)
		}
		posts = append(posts, article)
	}


	if err := t.ExecuteTemplate(w, "index", posts); err != nil {
		log.Printf("render index template: %v", err)
	}
}
func save_article(w http.ResponseWriter, r *http.Request) {
	title := r.FormValue("title")
	anons := r.FormValue("anons")
	full_text := r.FormValue("full_text")

	if title == "" || anons == "" || full_text == "" {
		fmt.Fprintf(w, "Incorrect data")
	}else{

		db, err := sql.Open("mysql", "root:root@tcp(127.0.0.1:3306)/golang")
		if err != nil {
			panic(err)
		}
		defer db.Close()

		insert, err := db.Query(fmt.Sprintf("INSERT INTO articles (title, anons, full_text) VALUES ('%s','%s','%s')", title, anons, full_text))
		if err != nil {
			panic(err)
		}
		defer insert.Close()
		http.Redirect(w,r, "/", http.StatusSeeOther)
	}
}
func show_post(w http.ResponseWriter, r *http.Request){
	t, err := template.ParseFS(templateFiles, "templates/post.html", "templates/head.html", "templates/footer.html")
	if err != nil {
		log.Printf("parse index template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	db, err := sql.Open("mysql", "root:root@tcp(127.0.0.1:3306)/golang")
		if err != nil {
			panic(err)
		}
		defer db.Close()
    vars := mux.Vars(r)
	w.WriteHeader(http.StatusOK)
	res, err:= db.Query(fmt.Sprintf("SELECT * FROM articles WHERE id = '%s'", vars["id"]))
	if err != nil {
		panic(err)
	}
	var article Article
	for res.Next(){
		
		err = res.Scan(&article.Id, &article.Title, &article.Anons, &article.FullText )
		if err != nil {
		panic(err)
		}
	}
	if err := t.ExecuteTemplate(w, "post", article); err != nil {
		log.Printf("render post template: %v", err)
	}
}
func handleFunc() error {
	rtr := mux.NewRouter()
	rtr.HandleFunc("/", index).Methods("GET")
	rtr.HandleFunc("/create", create).Methods("GET")
	rtr.HandleFunc("/save_article", save_article).Methods("POST")
	rtr.HandleFunc("/post/{id:[0-9]+}", show_post).Methods("GET")
	http.Handle("/", rtr)
	log.Println("serving on http://localhost:8081")
	return http.ListenAndServe(":8081", nil)
}

func main() {
	if err := handleFunc(); err != nil {
		log.Fatal(err)
	}
}
