package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"net/http"

	"github.com/mannders00/ezauth"
	_ "modernc.org/sqlite"
)

func main() {
	mux := http.NewServeMux()

	// Setup SQLite (in theory, any `*sql.DB`)
	db, err := sql.Open("sqlite", "file:data.db")
	if err != nil {
		panic(err)
	}

	cfg := ezauth.Config{DB: db}

	auth, err := ezauth.NewAuth(&cfg)

	auth.RegisterRoutes(mux)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		tmpl := template.Must(template.New("").ParseFiles("internal/index.html"))
		tmpl.ExecuteTemplate(w, "index.html", nil)
	})

	mux.Handle("/profile", auth.SessionMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := map[string]interface{}{
			"User": auth.GetCurrentUser(r),
		}
		tmpl := template.Must(template.New("").ParseFiles("internal/profile.html"))
		tmpl.ExecuteTemplate(w, "profile.html", data)
	})))

	fmt.Println("Listening on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		panic(err)
	}
}
