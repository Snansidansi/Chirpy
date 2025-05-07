package main

import "net/http"

func main() {
	serveMux := http.NewServeMux()
	httpServer := http.Server{
		Handler: serveMux,
		Addr:    ":8080",
	}

	httpServer.ListenAndServe()
}
