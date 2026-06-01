package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/generative-ai-go/genai"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type CacheKey struct {
	FileDigest string
	Question   string
}

func getDigestFromBytes(content []byte) string {
	hash := sha256.Sum256(content)
	return hex.EncodeToString(hash[:])
}

// Keeping this for backwards compatibility with tests if needed
func getFileDigest(fp string) (string, error) {
	content, err := os.ReadFile(fp)
	if err != nil {
		return "", err
	}
	return getDigestFromBytes(content), nil
}

func RegisterTools(s *server.MCPServer, cache *lru.Cache[CacheKey, string], client *genai.Client) {
	askFaqTool := mcp.NewTool("ask_faq",
		mcp.WithDescription("Ask a FAQ about a specific file. The cache will return an existing answer if one exists, otherwise it will generate a new one using Gemini and save it."),
		mcp.WithString("filepath",
			mcp.Required(),
			mcp.Description("The local file path the FAQ belongs to"),
		),
		mcp.WithString("question",
			mcp.Required(),
			mcp.Description("The question"),
		),
	)

	s.AddTool(askFaqTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		fp, err := request.RequireString("filepath")
		if err != nil {
			return mcp.NewToolResultError("filepath must be a string"), nil
		}
		question, err := request.RequireString("question")
		if err != nil {
			return mcp.NewToolResultError("question must be a string"), nil
		}

		cleanFp := filepath.Clean(fp)

		// 1. Read file and compute digest
		content, err := os.ReadFile(cleanFp)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to read file: %v", err)), nil
		}
		digest := getDigestFromBytes(content)

		// 2. Fetch existing questions for this digest
		keys := cache.Keys()
		var existingQuestions []string
		for _, k := range keys {
			if k.FileDigest == digest {
				existingQuestions = append(existingQuestions, k.Question)
			}
		}

		model := client.GenerativeModel("gemini-1.5-flash")

		// 3. Check for semantic matches if there are any existing cached questions
		if len(existingQuestions) > 0 {
			matchPrompt := fmt.Sprintf("I have a new question: '%s'. Does this question have the EXACT SAME semantic meaning as any of these questions?\n", question)
			for i, q := range existingQuestions {
				matchPrompt += fmt.Sprintf("%d. %s\n", i+1, q)
			}
			matchPrompt += "If there is a match, reply ONLY with the exact matching question string from the list. If there is no match, reply ONLY with 'NO MATCH'."

			resp, err := model.GenerateContent(ctx, genai.Text(matchPrompt))
			if err == nil && len(resp.Candidates) > 0 && resp.Candidates[0].Content != nil && len(resp.Candidates[0].Content.Parts) > 0 {
				var matchResult string
				if textPart, ok := resp.Candidates[0].Content.Parts[0].(genai.Text); ok {
					matchResult = strings.TrimSpace(string(textPart))
				}

				if matchResult != "" && matchResult != "NO MATCH" {
					// Find it in our list to be safe
					for _, q := range existingQuestions {
						if q == matchResult {
							if answer, ok := cache.Get(CacheKey{FileDigest: digest, Question: q}); ok {
								return mcp.NewToolResultText(fmt.Sprintf("[Cache Hit] %s", answer)), nil
							}
						}
					}
				}
			}
		}

		// 4. Generate new answer
		genPrompt := fmt.Sprintf("Answer the following question based on the provided document content.\nQuestion: %s\n\nDocument Content:\n%s", question, string(content))
		resp, err := model.GenerateContent(ctx, genai.Text(genPrompt))
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to generate answer: %v", err)), nil
		}

		var answer string
		if len(resp.Candidates) > 0 && resp.Candidates[0].Content != nil && len(resp.Candidates[0].Content.Parts) > 0 {
			if textPart, ok := resp.Candidates[0].Content.Parts[0].(genai.Text); ok {
				answer = string(textPart)
			}
		}

		if answer == "" {
			return mcp.NewToolResultError("failed to extract answer from Gemini response (possibly due to safety filters or empty content)"), nil
		}

		// Save to cache
		cache.Add(CacheKey{FileDigest: digest, Question: question}, answer)

		return mcp.NewToolResultText(fmt.Sprintf("[Generated] %s", answer)), nil
	})

	clearCacheTool := mcp.NewTool("clear_faq_cache",
		mcp.WithDescription("Clear all entries from the FAQ cache"),
	)

	s.AddTool(clearCacheTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		cache.Purge()
		return mcp.NewToolResultText("Cache cleared"), nil
	})
}
