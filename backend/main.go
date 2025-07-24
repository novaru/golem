package main

import (
	"flag"
	"fmt"
	"net/http"
)

func main() {
	port := flag.Int("port", 8001, "Port to listen on")
	name := flag.String("name", "backend1", "Backend name for responses")
	flag.Parse()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "%s responding to %s\n", *name, r.URL.Path)
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "ok")
	})

	fmt.Printf("Starting %s on port %d...\n", *name, *port)
	http.ListenAndServe(fmt.Sprintf(":%d", *port), nil)
}
