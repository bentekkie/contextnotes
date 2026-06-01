package main

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"testing"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetFileDigest(t *testing.T) {
	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "testfile")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	content := []byte("hello world")
	_, err = tmpFile.Write(content)
	require.NoError(t, err)
	tmpFile.Close()

	// Compute expected digest
	hash := sha256.Sum256(content)
	expectedDigest := hex.EncodeToString(hash[:])

	// Call getFileDigest
	digest, err := getFileDigest(tmpFile.Name())
	require.NoError(t, err)
	assert.Equal(t, expectedDigest, digest)
}

func TestCacheLogicWithDigest(t *testing.T) {
	cache, err := lru.New[CacheKey, string](100)
	require.NoError(t, err)

	digest := "some-fake-digest"
	key := CacheKey{FileDigest: digest, Question: "What is this?"}
	cache.Add(key, "It is a test.")

	val, ok := cache.Get(key)
	assert.True(t, ok)
	assert.Equal(t, "It is a test.", val)

	// Test cache keys lookup
	keys := cache.Keys()
	assert.Len(t, keys, 1)
	assert.Equal(t, digest, keys[0].FileDigest)

	cache.Purge()
	_, ok = cache.Get(key)
	assert.False(t, ok)
}
