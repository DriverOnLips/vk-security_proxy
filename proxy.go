package main

import (
	"fmt"
	"io"
	"net/http"
)

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

func handleRequest(w http.ResponseWriter, r *http.Request) {
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

	// copying headers from proxy response to src response
	copyResponseHeaders(response, w)

	// reading the body of the proxy response
	body, err := io.ReadAll(response.Body)
	if err != nil {
		http.Error(w, "Error reading proxy response body", http.StatusInternalServerError)
		return
	}

	// writing the body to the main response
	w.Write(body)
}


func main() {
	// creating server
	server := http.Server{
		Addr:    ":8080",
		Handler: http.HandlerFunc(handleRequest),
	}

	fmt.Println("server started on :8080")

	// starting server
	err := server.ListenAndServe()
	if err != nil {
		fmt.Errorf("Error starting proxy server: ", err)
	}
	
}