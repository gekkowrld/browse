package src

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"text/template"
)

type HomeSt struct {
	Name string
	Tag  string
	Dirs string
}

func Home(w http.ResponseWriter, r *http.Request) {
	config := *SetConfig()
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	data, err := templates.ReadFile("home.tmpl")
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	headerTmplData, err := templates.ReadFile("header.tmpl")
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	tmpl, err := template.New("index").Parse(string(data))
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	_, err = tmpl.New("header").Parse(string(headerTmplData))
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	var fid strings.Builder
	for _, file := range config.Directories {
		fn := filepath.Base(file)
		fid.WriteString(fmt.Sprintf(`<p><a href="/code/%s">%s</a> <span>%s</span></p>`, fn, fn, file))
	}

	var buf bytes.Buffer
	err = tmpl.ExecuteTemplate(&buf, "index", HomeSt{Name: config.Name, Tag: config.Tag, Dirs: fid.String()})
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.Write(buf.Bytes())
}
