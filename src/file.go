package src

import (
	"bytes"
	"github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"log"
	"os"
	"text/template"
)

type FileSt struct {
	Title   string
	Content string
	Navbar  string
}

func renderFile(filename, relpath string) []byte {
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
				return NotFound()
			}
			return fileContent
		}
	}

	if !isb {
		fileContent, err = os.ReadFile(filename)
		if err != nil {
			log.Println(err)
			return NotFound()
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
		return NotFound()
	}

	style := styles.Get("catppuccin-mocha")
	if style == nil {
		style = styles.Fallback
	}

	// Use chroma to highlight the code
	formatter := html.New(html.Standalone(true), html.WithClasses(true), html.WithLineNumbers(true), html.WithLinkableLineNumbers(true, "L"), html.WrapLongLines(true))
	var highlighted bytes.Buffer
	err = formatter.Format(&highlighted, style, iterator)
	if err != nil {
		log.Println(err)
		return NotFound()
	}

	// Read the templates
	fileTmplData, err := templates.ReadFile("file.tmpl")
	if err != nil {
		log.Println(err)
		return NotFound()
	}

	headerTmplData, err := templates.ReadFile("header.tmpl")
	if err != nil {
		log.Println(err)
		return NotFound()
	}

	// Parse both templates together
	tmpl, err := template.New("file").Parse(string(fileTmplData))
	if err != nil {
		log.Println(err)
		return NotFound()
	}

	_, err = tmpl.New("header").Parse(string(headerTmplData))
	if err != nil {
		log.Println(err)
		return NotFound()
	}

	var buf bytes.Buffer
	err = tmpl.ExecuteTemplate(&buf, "file", FileSt{Title: relpath, Navbar: navbar, Content: highlighted.String()})
	if err != nil {
		log.Println(err)
		return NotFound()
	}

	return buf.Bytes()

}
