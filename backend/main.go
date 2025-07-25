package main

import (
	"flag"
	"fmt"
	"net/http"
	"time"
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

	http.HandleFunc("/stream", func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)

		for {
			time.Sleep(200 * time.Millisecond)
			fmt.Fprintf(w, "%s streaming response at %s\n", *name, time.Now().Format(time.DateTime))
			flusher.Flush()
		}
	})

	fmt.Printf("Starting %s on port %d...\n", *name, *port)

	err := http.ListenAndServe(fmt.Sprintf(":%d", *port), nil)
	if err != nil {
		fmt.Printf("Server failed to start: %v\n", err)
	}
}
