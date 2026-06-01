package main

import (
	"testing"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/mark3labs/mcp-go/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCacheToolsDirectly(t *testing.T) {
	cache, err := lru.New[CacheKey, string](100)
	require.NoError(t, err)

	s := server.NewMCPServer("faq-cache", "1.0.0")
	RegisterTools(s, cache)

    // Test adding FAQ manually via cache map checking
	key := CacheKey{Filepath: "README.md", Question: "What is this?"}
    cache.Add(key, "It is a readme.")

    val, ok := cache.Get(key)
    assert.True(t, ok)
    assert.Equal(t, "It is a readme.", val)

	cache.Purge()
	_, ok = cache.Get(key)
	assert.False(t, ok)
}
