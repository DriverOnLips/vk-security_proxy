package main

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
)

func handleConnect(w http.ResponseWriter, r *http.Request) {
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}

	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer clientConn.Close()

	hostPort := r.URL.Host
	host := strings.Split(hostPort, ":")[0]
	port := strings.Split(hostPort, ":")[1]

	fmt.Fprintf(clientConn, "HTTP/1.1  200 Connection established\r\n\r\n")

	serverConn, err := net.Dial("tcp", host+":"+port)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer serverConn.Close()

	go func() {
		io.Copy(serverConn, clientConn)
	}()
	io.Copy(clientConn, serverConn)
}

func main() {
	http.HandleFunc("/", handleConnect)

	fmt.Println("Starting server on :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Printf("Error starting proxy server: %v\n", err)
	}
}
