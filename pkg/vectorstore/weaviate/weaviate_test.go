package weaviate_test

import (
	"context"
	"testing"

	"github.com/Ingenimax/agent-sdk-go/pkg/interfaces"
	"github.com/Ingenimax/agent-sdk-go/pkg/multitenancy"
	weaviatestore "github.com/Ingenimax/agent-sdk-go/pkg/vectorstore/weaviate"
)

func setupTestClient(t *testing.T) *interfaces.VectorStoreConfig {
	return &interfaces.VectorStoreConfig{
		Host:   "localhost:8080",
		APIKey: "test-key",
	}
}

func TestStore(t *testing.T) {
	config := setupTestClient(t)
	store := weaviatestore.New(config)

	ctx := multitenancy.WithOrgID(context.Background(), "test-org")

	// Test storing documents
	docs := []interfaces.Document{
		{
			ID:      "doc1",
			Content: "This is a test document",
			Metadata: map[string]interface{}{
				"source": "test",
			},
		},
		{
			ID:      "doc2",
			Content: "This is another test document",
			Metadata: map[string]interface{}{
				"source": "test",
			},
		},
	}

	err := store.Store(ctx, docs)
	if err != nil {
		t.Fatalf("Failed to store documents: %v", err)
	}

	// Test searching
	results, err := store.Search(ctx, "test document", 2)
	if err != nil {
		t.Fatalf("Failed to search: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	// Test getting documents
	retrieved, err := store.Get(ctx, []string{"doc1"})
	if err != nil {
		t.Fatalf("Failed to get document: %v", err)
	}

	if len(retrieved) != 1 {
		t.Fatalf("Expected 1 document, got %d", len(retrieved))
	}

	if retrieved[0].Content != docs[0].Content {
		t.Errorf("Expected content %q, got %q", docs[0].Content, retrieved[0].Content)
	}

	// Test deleting
	err = store.Delete(ctx, []string{"doc1", "doc2"})
	if err != nil {
		t.Fatalf("Failed to delete documents: %v", err)
	}

	// Verify deletion
	results, err = store.Search(ctx, "test document", 2)
	if err != nil {
		t.Fatalf("Failed to search: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Expected 0 results after deletion, got %d", len(results))
	}
}
