package main

import (
	"database/sql"
	"embed"
	"fmt"
	"html/template"
	"net/http"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

//go:embed static/*
var fs embed.FS
var db *sql.DB

func main() {

	// load templates
	tmpl, err := template.ParseFS(fs, "static/templates/*")
	if err != nil {
		panic(err)
	}

	// setup sqlite
	db, err = sql.Open("sqlite", "file:data.db")
	if err != nil {
		panic(err)
	}

	createTableSQL := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		email TEXT NOT NULL UNIQUE,
		password TEXT NOT NULL
	);`

	_, err = db.Exec(createTableSQL)
	if err != nil {
		panic(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "hello")
	})
	mux.HandleFunc("POST /register", registerHandler)
	mux.HandleFunc("GET /register", renderTemplate(tmpl, "register.html", nil))

	mux.HandleFunc("POST /login", loginHandler)
	mux.HandleFunc("GET /login", renderTemplate(tmpl, "login.html", nil))

	fmt.Println("Listening on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		panic(err)
	}
}

func renderTemplate(tmpl *template.Template, view string, data interface{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tmpl.ExecuteTemplate(w, view, data)
	}
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	password := r.FormValue("password")
	password_confirm := r.FormValue("password_confirm")

	if password != password_confirm {
		http.Error(w, "Passwords don't match", http.StatusBadRequest)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = db.Exec("INSERT INTO users (email, password) VALUES ($1, $2)", email, string(hashedPassword))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Println("User created")
	w.WriteHeader(http.StatusCreated)

}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	password := r.FormValue("password")

	var storedPassword string
	err := db.QueryRow("SELECT password FROM users WHERE email=($1)", email).Scan(&storedPassword)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(storedPassword), []byte(password))
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	fmt.Println("User verified")
	w.WriteHeader(http.StatusCreated)
}
