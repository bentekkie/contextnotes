package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/google/generative-ai-go/genai"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/mark3labs/mcp-go/server"
	"google.golang.org/api/option"
)

func main() {
	modelFlag := flag.String("model", "gemini-pro", "The Gemini model to use")
	flag.Parse()

	ctx := context.Background()

	// Initialize the LRU cache (size 100)
	cache, err := lru.New[CacheKey, string](100)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize cache: %v\n", err)
		os.Exit(1)
	}

	// Initialize Gemini client
	apiKey := os.Getenv("GEMINI_API_KEY")
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize Gemini client: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	s := server.NewMCPServer("faq-cache", "1.0.0")

	RegisterTools(s, cache, client, *modelFlag)

	stdioServer := server.NewStdioServer(s)
	if err := stdioServer.Listen(context.Background(), os.Stdin, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
	}
}
