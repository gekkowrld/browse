package src

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/http"
)

type ErrorSt struct {
	Name      string
	Tag       string
	Error     string
	ErrorCode int
	Reason    string
	URL       string
	Help      string
}

var (
	tmplError  *template.Template
	tmplHeader *template.Template
)

func init() {
	var err error
	fileTmplData, err := templates.ReadFile("weberror.tmpl")
	if err != nil {
		log.Fatal("Error reading weberror.tmpl: ", err)
	}

	headerTmplData, err := templates.ReadFile("header.tmpl")
	if err != nil {
		log.Fatal("Error reading header.tmpl: ", err)
	}

	tmplError, err = template.New("weberror").Parse(string(fileTmplData))
	if err != nil {
		log.Fatal("Error parsing weberror.tmpl: ", err)
	}

	tmplHeader, err = tmplError.New("header").Parse(string(headerTmplData))
	if err != nil {
		log.Fatal("Error parsing header.tmpl: ", err)
	}
}

func NotFound(w http.ResponseWriter, r *http.Request, reason string) {
	Error(w, r, http.StatusNotFound, reason)
}

func InternalError(w http.ResponseWriter, r *http.Request, reason string) {
	Error(w, r, http.StatusInternalServerError, reason)
}

func Error(w http.ResponseWriter, r *http.Request, status int, reason string) {
	// Set the HTTP status code to the error status
	w.WriteHeader(status)

	// Read and parse the template files
	fileTmplData, err := templates.ReadFile("weberror.tmpl")
	if err != nil {
		log.Println("Error reading weberror.tmpl: ", err)
		fatalError(w, r)
		return
	}

	headerTmplData, err := templates.ReadFile("header.tmpl")
	if err != nil {
		log.Println("Error reading header.tmpl: ", err)
		fatalError(w, r)
		return
	}

	tmpl, err := template.New("weberror").Parse(string(fileTmplData))
	if err != nil {
		log.Println("Error parsing weberror.tmpl: ", err)
		fatalError(w, r)
		return
	}

	_, err = tmpl.New("header").Parse(string(headerTmplData))
	if err != nil {
		log.Println("Error parsing header.tmpl: ", err)
		fatalError(w, r)
		return
	}

	url := r.URL.String()
	var buf bytes.Buffer
	err = tmpl.ExecuteTemplate(&buf, "weberror", ErrorSt{
		Error:     http.StatusText(status),
		Tag:       config.Tag,
		Name:      config.Name,
		ErrorCode: status,
		Reason:    reason,
		URL:       url,
		Help:      `Go back to Home`,
	})

	if err != nil {
		log.Println("Error executing template: ", err)
		fatalError(w, r)
		return
	}

	// Set the Content-Type header to indicate HTML content
	w.Header().Set("Content-Type", "text/html")
	// Write the response body
	w.Write(buf.Bytes())
}

// fatalError is a last resort for when error templates fail
// fatalError handles critical errors that occur when error templates fail.
func fatalError(w http.ResponseWriter, r *http.Request) {
	// Construct the error page HTML
	errStr := fmt.Sprintf(`
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Fatal Error</title>
    <style>
        body {
            color: #fff;
            background-color: #f44336;
            font-family: Arial, sans-serif;
            text-align: center;
            padding: 50px;
        }
        h1 {
            color: #fff;
        }
    </style>
</head>
<body>
    <h1>Fatal Error</h1>
    <p>A critical error occurred while processing your request.</p>
	<p>A meaningful error could not be displayed, sorry for that :(</p>
	<p>Don't worry, go back to <a href="/">Home</a> and try your luck!</p>
    <p>URL: %s</p>
</body>
</html>
`, r.URL.String())

	// Set the content type to HTML and write the error page to the response
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusInternalServerError) // Set appropriate HTTP status code
	w.Write([]byte(errStr))
}
