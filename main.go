package main

import (
	"context"
	"fmt"
	"os"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	cache, err := lru.New[CacheKey, string](100)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize cache: %v\n", err)
		os.Exit(1)
	}

	s := server.NewMCPServer("faq-cache", "1.0.0")

    RegisterTools(s, cache)

	stdioServer := server.NewStdioServer(s)
	if err := stdioServer.Listen(context.Background(), os.Stdin, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
	}
}
