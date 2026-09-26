package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net/http"
)

func main() {
	port := flag.Int("port", 9001, "port to listen on")
	flag.Parse()

	addr := fmt.Sprintf(":%d", *port)
	msg := fmt.Sprintf("hello from server %s", addr)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, msg)
	})
	slog.Info("mock server listening", "addr", addr)
	slog.Error("server stopped", "err", http.ListenAndServe(addr, nil))
}
