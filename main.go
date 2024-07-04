package main

import (
	"database/sql"
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"path/filepath"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

//go:embed resources
var embedFS embed.FS
var db *sql.DB
var tmpl map[string]*template.Template

// Parse templates into `tmpl` and initialize `db`
func init() {

	// load templates
	tmpl = make(map[string]*template.Template)

	var err error
	// Load templates
	err = fs.WalkDir(embedFS, "resources/views", func(viewPath string, d fs.DirEntry, err error) error {
		if !d.IsDir() {
			viewPathBase := filepath.Base(viewPath)
			tmpl[viewPathBase] = template.Must(template.ParseFS(embedFS, viewPath, "resources/templates/*"))
		}
		return nil
	})
	if err != nil {
		panic(err)
	}

	// Setup SQLite (in theory, any `*sql.DB`)
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
}

func main() {
	defer db.Close()
	mux := http.NewServeMux()

	// Register new user
	mux.HandleFunc("POST /register", registerHandler)
	mux.HandleFunc("GET /register", renderView("register.html", nil))

	// Log In
	mux.HandleFunc("POST /login", loginHandler)
	mux.HandleFunc("GET /login", renderView("login.html", nil))

	mux.Handle("/resources/", http.FileServerFS(embedFS))

	fmt.Println("Listening on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		panic(err)
	}
}

func renderView(view string, data interface{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tmpl[view].ExecuteTemplate(w, "base", data)
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

	if len(password) <= 8 {
		http.Error(w, "Password should be greater than ", http.StatusBadRequest)
		return
	}

	// other checks

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

	tmpl["login.html"].ExecuteTemplate(w, "content", nil)
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

	fmt.Fprint(w, "all good man you're in")
}
