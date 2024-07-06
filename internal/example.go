package main

import (
	"database/sql"
	"fmt"
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

	auth, err := ezauth.NewAuth(db)

	auth.RegisterRoutes(mux)

	fmt.Println("Listening on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		panic(err)
	}
}
