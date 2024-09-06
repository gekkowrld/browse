package disp

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
	fileTmplData, err := templates.ReadFile("templates/weberror.tmpl")
	if err != nil {
		log.Println("Error reading weberror.tmpl: ", err)
		fatalError(w, r)
		return
	}

	headerTmplData, err := templates.ReadFile("templates/header.tmpl")
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
	errStr := fmt.Sprintf(`
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Oops!</title>
    <style>
@import url('https://fonts.googleapis.com/css2?family=Kanit:ital,wght@0,100;0,200;0,300;0,400;0,500;0,600;0,700;0,800;0,900;1,100;1,200;1,300;1,400;1,500;1,600;1,700;1,800;1,900&display=swap');
        body {
            color: #fff;
            background-color: #f44336;
            text-align: center;
            padding: 50px;
  			font-family: "Kanit", sans-serif;
			font-weight: 400;
			font-style: normal;
        }
        h1 {
            color: #fff;
        }
    </style>
</head>
<body>
    <h1>Something Went Hooooribly Wrong</h1>
    <p>We encountered an issue while processing your request.</p>
    <p>Unfortunately, we couldn't display a detailed error message. Please try again later.</p>
    <p>Feel free to return to the <a href="/">Home</a> page and continue browsing.</p>
    <p>URL: %s</p>
</body>
</html>
`, r.URL.String())

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte(errStr))
}
