package ezauth

import (
	"crypto/rand"
	"database/sql"
	"embed"
	"encoding/base64"
	"html/template"
	"io/fs"
	"net/http"
	"path/filepath"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type Config struct{}

//go:embed resources
var embedFS embed.FS

type Auth struct {
	db   *sql.DB
	tmpl map[string]*template.Template
}

func NewAuth(db *sql.DB) (*Auth, error) {
	a := &Auth{
		db:   db,
		tmpl: make(map[string]*template.Template),
	}

	// Here we walk over all views/*, set the view name as a key of `tmpl`,
	// Then parse all templates/* into that value of *template.Template.
	// This approach lets us define multiple views extending shared templates
	// while also preloading templates into memory.
	//
	// tmpl["login.html"].ExecuteTemplate(w, "content", nil)	<- Renders just the content block
	// tmpl["login.html"].ExecuteTemplate(w, "base", nil)		<- Renders the whole page
	var err error
	err = fs.WalkDir(embedFS, "resources/views", func(viewPath string, d fs.DirEntry, err error) error {
		if !d.IsDir() {
			viewPathBase := filepath.Base(viewPath)
			a.tmpl[viewPathBase] = template.Must(template.ParseFS(embedFS, viewPath, "resources/templates/*"))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	createTableSQL := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		email TEXT NOT NULL UNIQUE,
		password TEXT NOT NULL
	);
	CREATE TABLE IF NOT EXISTS sessions (
		session_id VARCHAR(255) PRIMARY KEY,
		user_email INTEGER,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		expires_at TIMESTAMP,
		FOREIGN KEY (user_email) REFERENCES users(email) ON DELETE CASCADE
	);`

	_, err = db.Exec(createTableSQL)
	if err != nil {
		return nil, err
	}

	return a, nil
}

func (a *Auth) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /auth/register", a.registerHandler)
	mux.HandleFunc("GET /auth/register", a.renderView("register.html", nil))
	mux.HandleFunc("POST /auth/login", a.loginHandler)
	mux.HandleFunc("GET /auth/login", a.renderView("login.html", nil))
	mux.Handle("/auth/resources/", http.StripPrefix("/auth/", http.FileServerFS(embedFS)))
}

func (a *Auth) renderView(view string, data interface{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		a.tmpl[view].ExecuteTemplate(w, "base", data)
	}
}

func (a *Auth) registerHandler(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	password := r.FormValue("password")
	password_confirm := r.FormValue("password_confirm")

	if password != password_confirm {
		http.Error(w, "Passwords don't match", http.StatusBadRequest)
		return
	}

	if len(password) <= 8 {
		http.Error(w, "Password must be at least 8 characters", http.StatusBadRequest)
		return
	}

	// other checks

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = a.db.Exec("INSERT INTO users (email, password) VALUES ($1, $2)", email, string(hashedPassword))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// "Redirect" to login by returning "content" from login.html, which is swapped using htmx
	a.tmpl["login.html"].ExecuteTemplate(w, "content", nil)
}

func (a *Auth) loginHandler(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	password := r.FormValue("password")

	var storedPassword string
	err := a.db.QueryRow("SELECT password FROM users WHERE email=($1)", email).Scan(&storedPassword)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(storedPassword), []byte(password))
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	session_id := generateSessionID()
	expiresAt := time.Now().Add(time.Hour * 48)

	_, err = a.db.Exec("INSERT INTO sessions (session_id, user_email, expires_at) VALUES ($1, $2, $3)", session_id, email, expiresAt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    session_id,
		HttpOnly: true,
		Path:     "/",
	})
}

func (a *Auth) SessionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session_id")
		if err != nil {
			if err == http.ErrNoCookie {
				http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
				return
			}
			http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
			return
		}

		var email string
		err = a.db.QueryRow("SELECT user_email FROM sessions WHERE session_id=($1)", cookie.Value).Scan(&email)
		if err != nil {
			http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func generateSessionID() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}
