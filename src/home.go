package src

import (
	"bytes"
	"log"
	"net/http"
	"text/template"
)

func Home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		w.Write(NotFound())
		return
	}

	data, err := templates.ReadFile("home.tmpl")
	if err != nil {
		log.Println(err)
		w.Write(NotFound())
	}

	headerTmplData, err := templates.ReadFile("header.tmpl")
	if err != nil {
		log.Println(err)
		w.Write(NotFound())
		return
	}

	tmpl, err := template.New("index").Parse(string(data))
	if err != nil {
		log.Println(err)
		w.Write(NotFound())
		return
	}

	_, err = tmpl.New("header").Parse(string(headerTmplData))
	if err != nil {
		log.Println(err)
		w.Write(NotFound())
		return
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, nil)
	if err != nil {
		log.Println(err)
		w.Write(NotFound())
		return
	}

	w.Write(buf.Bytes())
}
