package main

import (
	"codeberg.org/gekkowrld/browse/src"
	_ "embed"
	"fmt"
	"log"
	"net/http"
	"strings"
)

//go:embed favicon.ico
var favicon []byte

func main() {
	cfg := src.SetConfig()
	port := cfg.Port
	host := cfg.Host
	open_at := fmt.Sprintf("%s:%d", strings.TrimSuffix(host, ":"), port)
	http.HandleFunc("/", src.Home)
	http.HandleFunc("/code/", src.Code)
	http.HandleFunc("/media/", src.Media)
	http.HandleFunc("/search", src.Search)
	http.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.Write(favicon)
	})
	log.Printf("`%s` - '%s' at %s", cfg.Name, cfg.Tag, open_at)
	log.Fatal(http.ListenAndServe(open_at, nil))
}
