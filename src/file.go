package src

import (
	"bytes"
	"fmt"
	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	gm "github.com/dustin/go-humanize"
	"github.com/go-enry/go-enry/v2"
	gohtml "html"
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
	Lines    int
	Text     bool
	Media    string
}

func renderFile(w http.ResponseWriter, r *http.Request, filename, relpath string) {
	var fileContent []byte
	var err error
	var fileSize int
	isb := isBinary(filename)

	if isb {
		media_type, ok := isViewableInBrowser(filename)
		if !ok {
			fileContent = []byte("The file is a binary file, can't display it")
		} else {
			if media_type == "pdf" {
				renderMedia(media_type, filename, w, r)
			} else {
				renderMedia(media_type, relpath, w, r)
			}
			return
		}
	} else {
		fileContent, err = os.ReadFile(filename)
		if err != nil {
			log.Println(err)
			NotFound(w, r, fmt.Sprintf("%s not found!", filename))
			return
		}
		fileSize = len(fileContent)
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

	iter, _ := lexer.Tokenise(nil, string(fileContent))
	iterLen := len(chroma.SplitTokensIntoLines(iter.Tokens()))

	lang := enry.GetLanguage(filename, fileContent)
	if lang == "" {
		lang = fmt.Sprintf("<span class='no_lang'>%s</span>", config.Name)
	}
	var buf bytes.Buffer
	err = tmpl.ExecuteTemplate(&buf, "file", FileSt{
		Title:    relpath,
		Language: lang,
		Navbar:   navbar,
		Content:  highlighted.String(),
		Tag:      config.Tag,
		Size:     gm.Bytes(uint64(fileSize)),
		Lines:    iterLen,
		Text:     true,
	})
	if err != nil {
		log.Println(err)
		InternalError(w, r, "Couldn't execute 'file' template")
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.Write(buf.Bytes())
}

func renderMedia(media_type, filename string, w http.ResponseWriter, r *http.Request) {
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

	var media string

	switch media_type {
	case "img":
		media = fmt.Sprintf(`<img src="/media/%s" alt="%s">`, gohtml.EscapeString(filename), gohtml.EscapeString(filename))
	case "audio":
		media = fmt.Sprintf(`<audio controls><source src="/media/%s" type="audio/mpeg">Your browser does not support the audio element.</audio>`, gohtml.EscapeString(filename))
	case "video":
		media = fmt.Sprintf(`<video controls><source src="/media/%s" type="video/mp4">Your browser does not support the video tag.</video>`, gohtml.EscapeString(filename))
	case "pdf":
		pdf_c, err := os.ReadFile(filename)
		if err != nil {
			log.Println(err)
			InternalError(w, r, "pDF not found!")
			return
		}
		w.Header().Set("Content-Type", "application/pdf")
		w.Write(pdf_c)
		return
	default:
		media = `<p>Unsupported media type</p>`
	}

	var buf bytes.Buffer
	err = tmpl.ExecuteTemplate(&buf, "file", FileSt{
		Title:  filename,
		Navbar: navbar,
		Tag:    config.Tag,
		Text:   false,
		Media:  media,
	})
	if err != nil {
		log.Println(err)
		InternalError(w, r, "Couldn't execute 'file' template")
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.Write(buf.Bytes())
}
