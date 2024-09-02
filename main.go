package main

import (
	"codeberg.org/gekkowrld/browse/src"
	_ "embed"
	"log"
	"net/http"
)

//go:embed favicon.ico
var favicon []byte

func main() {
	open_at := "0.0.0.0:9789"
	http.HandleFunc("/", src.Home)
	http.HandleFunc("/code/", src.Code)
	http.HandleFunc("/search", src.Search)
	http.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.Write(favicon)
	})
	log.Println("Open at http://localhost",open_at)
	log.Fatal(http.ListenAndServe(open_at, nil))
}
