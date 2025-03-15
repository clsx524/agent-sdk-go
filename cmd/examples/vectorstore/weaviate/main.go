package main

import (
	"context"
	"fmt"

	"github.com/Ingenimax/agent-sdk-go/pkg/config"
	"github.com/Ingenimax/agent-sdk-go/pkg/embedding"
	"github.com/Ingenimax/agent-sdk-go/pkg/interfaces"
	"github.com/Ingenimax/agent-sdk-go/pkg/logging"
	"github.com/Ingenimax/agent-sdk-go/pkg/multitenancy"
	"github.com/Ingenimax/agent-sdk-go/pkg/vectorstore/weaviate"
	"github.com/google/uuid"
)

func main() {
	// Create a logger
	logger := logging.New()

	ctx := multitenancy.WithOrgID(context.Background(), "exampleorg")

	// Load configuration
	cfg := config.Get()

	// Initialize the OpenAIEmbedder with the API key and model from config
	embedder := embedding.NewOpenAIEmbedder(cfg.LLM.OpenAI.APIKey, cfg.LLM.OpenAI.EmbeddingModel)

	store := weaviate.New(
		&interfaces.VectorStoreConfig{
			Host:   cfg.VectorStore.Weaviate.Host,
			APIKey: cfg.VectorStore.Weaviate.APIKey,
			Scheme: cfg.VectorStore.Weaviate.Scheme,
		},
		weaviate.WithClassPrefix("TestDoc"),
		weaviate.WithEmbedder(embedder),
		weaviate.WithLogger(logger),
	)

	docs := []interfaces.Document{
		{
			ID:      uuid.New().String(),
			Content: "The quick brown fox jumps over the lazy dog",
			Metadata: map[string]interface{}{
				"source": "example",
				"type":   "pangram",
			},
		},
		{
			ID:      uuid.New().String(),
			Content: "To be or not to be, that is the question",
			Metadata: map[string]interface{}{
				"source": "example",
				"type":   "quote",
			},
		},
	}

	// Embedding generation
	for idx, doc := range docs {
		vector, err := embedder.Embed(ctx, doc.Content)
		if err != nil {
			logger.Error(ctx, "Embedding failed", map[string]interface{}{"error": err.Error()})
			return
		}
		docs[idx].Vector = vector
	}

	fmt.Println("Storing documents with embeddings...")
	if err := store.Store(ctx, docs); err != nil {
		logger.Error(ctx, "Failed to store documents", map[string]interface{}{"error": err.Error()})
		return
	}

	fmt.Println("Searching for 'fox jumps'...")
	results, err := store.Search(ctx, "fox jumps", 5, interfaces.WithEmbedding(true))
	if err != nil {
		logger.Error(ctx, "Search failed", map[string]interface{}{"error": err.Error()})
		return
	}

	if len(results) == 0 {
		fmt.Println("No results found with embedding search.")
	} else {
		fmt.Println("Search results:")
		for _, r := range results {
			fmt.Printf("- %s (score: %.2f)\n", r.Document.Content, r.Score)
		}
	}

	// Cleanup
	var ids []string
	for _, doc := range docs {
		ids = append(ids, doc.ID)
	}
	if err := store.Delete(ctx, ids); err != nil {
		logger.Error(ctx, "Cleanup failed", map[string]interface{}{"error": err.Error()})
		return
	}
	fmt.Println("Cleanup successful")
}
