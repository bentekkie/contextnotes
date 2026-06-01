package main

import (
	"context"
	"fmt"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type CacheKey struct {
	Filepath string
	Question string
}

func RegisterTools(s *server.MCPServer, cache *lru.Cache[CacheKey, string]) {
	addFaqTool := mcp.NewTool("add_faq",
		mcp.WithDescription("Add a FAQ to the cache"),
		mcp.WithString("filepath",
			mcp.Required(),
			mcp.Description("The local file path the FAQ belongs to"),
		),
		mcp.WithString("question",
			mcp.Required(),
			mcp.Description("The question"),
		),
		mcp.WithString("answer",
			mcp.Required(),
			mcp.Description("The answer"),
		),
	)

	s.AddTool(addFaqTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		filepath, err := request.RequireString("filepath")
		if err != nil {
			return mcp.NewToolResultError("filepath must be a string"), nil
		}
		question, err := request.RequireString("question")
		if err != nil {
			return mcp.NewToolResultError("question must be a string"), nil
		}
		answer, err := request.RequireString("answer")
		if err != nil {
			return mcp.NewToolResultError("answer must be a string"), nil
		}

		key := CacheKey{Filepath: filepath, Question: question}
		cache.Add(key, answer)

		return mcp.NewToolResultText(fmt.Sprintf("Added FAQ for %s: %s", filepath, question)), nil
	})

	getFaqTool := mcp.NewTool("get_faq",
		mcp.WithDescription("Get a FAQ from the cache"),
		mcp.WithString("filepath",
			mcp.Required(),
			mcp.Description("The local file path the FAQ belongs to"),
		),
		mcp.WithString("question",
			mcp.Required(),
			mcp.Description("The question"),
		),
	)

	s.AddTool(getFaqTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		filepath, err := request.RequireString("filepath")
		if err != nil {
			return mcp.NewToolResultError("filepath must be a string"), nil
		}
		question, err := request.RequireString("question")
		if err != nil {
			return mcp.NewToolResultError("question must be a string"), nil
		}

		key := CacheKey{Filepath: filepath, Question: question}
		if answer, ok := cache.Get(key); ok {
			return mcp.NewToolResultText(answer), nil
		}

		return mcp.NewToolResultError(fmt.Sprintf("FAQ not found for %s: %s", filepath, question)), nil
	})

	clearCacheTool := mcp.NewTool("clear_faq_cache",
		mcp.WithDescription("Clear all entries from the FAQ cache"),
	)

	s.AddTool(clearCacheTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		cache.Purge()
		return mcp.NewToolResultText("Cache cleared"), nil
	})
}
