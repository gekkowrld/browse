package src

import (
	"bytes"
	"fmt"
	"github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"log"
	"net/http"
	"os"
	"text/template"
)

type FileSt struct {
	Title    string
	Tag      string
	Content  string
	Navbar   string
	Language string
	Size     string
}

func renderFile(w http.ResponseWriter, r *http.Request, filename, relpath string) {
	var fileContent []byte
	var err error
	isb := isBinary(filename)

	if isb {
		if !isViewableInBrowser(filename) {
			fileContent = []byte("The file is a binary file, can't display it")
		} else {
			fileContent, err = os.ReadFile(filename)
			if err != nil {
				log.Println(err)
				NotFound(w, r, fmt.Sprintf("%s not found!", filename))
				return
			}
		}
	} else {
		fileContent, err = os.ReadFile(filename)
		if err != nil {
			log.Println(err)
			NotFound(w, r, fmt.Sprintf("%s not found!", filename))
			return
		}
	}

	lexer := lexers.Match(filename)
	if lexer == nil {
		lexer = lexers.Analyse(string(fileContent))
		if lexer == nil {
			lexer = lexers.Fallback
		}
	}

	iterator, err := lexer.Tokenise(nil, string(fileContent))
	if err != nil {
		log.Println(err)
		NotFound(w, r, "Error tokenising the file content")
		return
	}

	style := styles.Get("catppuccin-mocha")
	if style == nil {
		style = styles.Fallback
	}

	// Use chroma to highlight the code
	formatter := html.New(html.WithLineNumbers(true), html.WithLinkableLineNumbers(true, "L"), html.WrapLongLines(true))
	var highlighted bytes.Buffer
	err = formatter.Format(&highlighted, style, iterator)
	if err != nil {
		log.Println(err)
		NotFound(w, r, "Error formatting the highlighted code")
		return
	}

	fileTmplData, err := templates.ReadFile("file.tmpl")
	if err != nil {
		log.Println(err)
		InternalError(w, r, "Error reading the file template")
		return
	}

	headerTmplData, err := templates.ReadFile("header.tmpl")
	if err != nil {
		log.Println(err)
		InternalError(w, r, "Error reading the header template")
		return
	}

	tmpl, err := template.New("file").Parse(string(fileTmplData))
	if err != nil {
		log.Println(err)
		InternalError(w, r, "Error parsing the file template")
		return
	}

	_, err = tmpl.New("header").Parse(string(headerTmplData))
	if err != nil {
		log.Println(err)
		InternalError(w, r, "Error parsing the header template")
		return
	}

	lang := fmt.Sprintf("%+v", lexer)
	var buf bytes.Buffer
	err = tmpl.ExecuteTemplate(&buf, "file", FileSt{Title: relpath, Language: lang, Navbar: navbar, Content: highlighted.String(), Tag: config.Tag})
	if err != nil {
		log.Println(err)
		InternalError(w, r, "Couldn't execute 'file' template")
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.Write(buf.Bytes())
}
