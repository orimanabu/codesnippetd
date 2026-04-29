package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
)

func main() {
	addr := flag.String("listen", ":8999", "listen address (host:port)")
	flag.StringVar(addr, "l", ":8999", "listen address (shorthand for -listen)")
	port := flag.Int("port", 0, "port number to listen on; overrides -addr when set")
	flag.IntVar(port, "p", 0, "port number to listen on (shorthand for -port)")
	noTreeSitter := flag.Bool("no-tree-sitter", false, "disable tree-sitter; use ctags parser only")
	// Kept for backwards compatibility: --tree-sitter is now a no-op since tree-sitter is on by default.
	_ = flag.Bool("tree-sitter", true, "use tree-sitter to resolve end lines (default; deprecated flag)")
	flag.Parse()

	useTreeSitter := !*noTreeSitter

	listenAddr := *addr
	if *port != 0 {
		listenAddr = fmt.Sprintf(":%d", *port)
	}

	mux := http.NewServeMux()
	registerHandlers(mux, useTreeSitter)

	log.Printf("listening on %s", listenAddr)
	if err := http.ListenAndServe(listenAddr, corsMiddleware(accessLog(mux))); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
