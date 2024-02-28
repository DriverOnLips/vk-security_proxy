package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	_ "github.com/lib/pq"
)

var db *sql.DB

func printRequest(r *http.Request) {
	fmt.Println("=========Request=========")
	fmt.Println(r.Method, r.RequestURI, r.Proto)
	fmt.Println("Host:", r.Host)
	fmt.Println("User-Agent:", r.UserAgent())
	fmt.Println("Accept:", r.Header.Get("Accept"))
	fmt.Println("Proxy-Connection:", r.Header.Get("Proxy-Connection"))
	fmt.Println()
}

func printResponse(response *http.Response) {
	fmt.Println("=========Response=========")
	fmt.Println(response.Proto, response.Status)
	fmt.Println("Server: ", response.Header.Get("Server"))
	fmt.Println("Date: ", response.Header.Get("Date"))
	fmt.Println("Content-Type: ", response.Header.Get("Content-Type"))
	fmt.Println("Content-Length: ", response.Header.Get("Content-Length"))
	fmt.Println("Connection: ", response.Header.Get("Connection"))
	fmt.Println("Location: ", response.Header.Get("Location"))
	fmt.Println()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		fmt.Println("Error reading response body")
	} else {
		fmt.Println(string(body))
		fmt.Println()
	}
}

func copyRequestHeaders(from *http.Request, to *http.Request) {
	for name, values := range from.Header {
		for _, value := range values {
			to.Header.Set(name, value)
		}
	}
}

func copyResponseHeaders(from *http.Response, to http.ResponseWriter) {
	for name, values := range from.Header {
		for _, value := range values {
			to.Header().Set(name, value)
		}
	}

	// set the status code from proxy response to src response
	to.WriteHeader(from.StatusCode)
}

func saveRequestToDB(r *http.Request) error {
    // Parse cookies
    cookies := make(map[string]interface{})
    for _, cookie := range r.Cookies() {
        cookies[cookie.Name] = cookie.Value
    }

    // Parse POST form
    err := r.ParseForm()
    if err != nil {
        return err
    }

    postParams := make(map[string]interface{})
    for key, values := range r.PostForm {
        if len(values) > 0 {
            postParams[key] = values[0]
        }
    }

    // Marshal request data
    getParams := make(map[string]interface{})
    for key, values := range r.URL.Query() {
        if len(values) > 0 {
            getParams[key] = values[0]
        }
    }

    headers := make(map[string]interface{})
    for key, values := range r.Header {
        if len(values) > 0 {
            headers[key] = values[0]
        }
    }

    // Marshal request data
    requestData, err := json.Marshal(map[string]interface{}{
        "method":      r.Method,
        "path":        r.URL.Path,
        "get_params":  getParams,
        "headers":     headers,
        "cookies":     cookies,
        "post_params": postParams,
    })
    if err != nil {
        return err
    }

		log.Print(requestData)

    // Insert request data into the database
    _, err = db.Exec("INSERT INTO requests (method, path, get_params, headers, cookies, post_params) VALUES ($1, $2, $3, $4, $5, $6)",
        r.Method, r.URL.Path, getParams, headers, cookies, postParams)
    return err
}


func saveResponseToDB(code int, message string, headers http.Header, body []byte) error {
    // Parse response headers
    headersMap := make(map[string]interface{})
    for key, values := range headers {
        if len(values) > 0 {
            headersMap[key] = values[0]
        }
    }

    // Insert response data into the database
    _, err := db.Exec("INSERT INTO responses (code, message, headers, body) VALUES ($1, $2, $3, $4)",
        code, message, headersMap, string(body))
    return err
}


func handleRequest(w http.ResponseWriter, r *http.Request) {
    // Saving request to DB
    err := saveRequestToDB(r)
    if err != nil {
        http.Error(w, "Error saving request to DB", http.StatusInternalServerError)
        return
    }

    // printing request
    printRequest(r)

    // deleting header Proxy-Connection
    r.Header.Del("Proxy-Connection")

    // replacing path with relative
    r.RequestURI = r.URL.Path

    // creating proxy request with the same method, url and body
    proxyReq, err := http.NewRequest(r.Method, r.URL.String(), r.Body)
    if err != nil {
        http.Error(w, "cant create proxy request", http.StatusInternalServerError)
        return
    }

    // copying headers from src request to proxy request
    copyRequestHeaders(r, proxyReq)

    // sending proxy request using transport
    response, err := http.DefaultTransport.RoundTrip(proxyReq)
    if err != nil {
        http.Error(w, "Error sending proxy request", http.StatusInternalServerError)
        return
    }
    defer response.Body.Close()

    // printing response
    printResponse(response)

    // reading the body of the proxy response
    body, err := io.ReadAll(response.Body)
    if err != nil {
        http.Error(w, "Error reading proxy response body", http.StatusInternalServerError)
        return
    }

    // Saving response data to DB
    err = saveResponseToDB(response.StatusCode, response.Status, response.Header, body)
    if err != nil {
        http.Error(w, "Error saving response to DB", http.StatusInternalServerError)
        return
    }


    // copying headers from proxy response to src response
    copyResponseHeaders(response, w)

    // writing the body to the main response
    w.Write(body)
}


func main() {
    // Connect to PostgreSQL
    connStr := fmt.Sprintf("host=db user=%s dbname=%s password=%s sslmode=disable", 
    os.Getenv("POSTGRES_USER"), 
    os.Getenv("POSTGRES_DB"), 
    os.Getenv("POSTGRES_PASSWORD"))

    var err error
    db, err = sql.Open("postgres", connStr)
    if err != nil {
        panic(err)
    }
    defer db.Close()


    // creating server
    server := http.Server{
        Addr:    ":8080",
        Handler: http.HandlerFunc(handleRequest),
    }

    fmt.Println("server started on :8080")

    // starting server
    err = server.ListenAndServe()
    if err != nil {
        fmt.Errorf("Error starting proxy server: ", err)
    }
}
