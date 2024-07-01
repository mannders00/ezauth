package main

import (
	"database/sql"
	"fmt"
	_ "modernc.org/sqlite"
	"net/http"
)

func main() {

	db, err := sql.Open("sqlite", "file:data.db")
	if err != nil {
		panic(err)
	}

	createTableSQL := `
	Create Table if not exists users (
		id integer primary key autoincrement,
		name text not null
	);`

	_, err = db.Exec(createTableSQL)
	if err != nil {
		panic(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "hello")
	})
	if err := http.ListenAndServe(":8080", mux); err != nil {
		panic(err)
	}
}
