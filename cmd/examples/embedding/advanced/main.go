package main

import (
	"context"
	"fmt"
	"math"

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

	// Initialize the OpenAIEmbedder with custom configuration
	embeddingConfig := embedding.DefaultEmbeddingConfig(cfg.LLM.OpenAI.EmbeddingModel)
	embeddingConfig.Dimensions = 1536 // Specify dimensions for more precise embeddings
	embeddingConfig.SimilarityMetric = "cosine"
	embeddingConfig.SimilarityThreshold = 0.6 // Set a similarity threshold

	embedder := embedding.NewOpenAIEmbedderWithConfig(cfg.LLM.OpenAI.APIKey, embeddingConfig)

	store := weaviate.New(
		&interfaces.VectorStoreConfig{
			Host:   cfg.VectorStore.Weaviate.Host,
			APIKey: cfg.VectorStore.Weaviate.APIKey,
			Scheme: cfg.VectorStore.Weaviate.Scheme,
		},
		weaviate.WithEmbedder(embedder),
		weaviate.WithClassPrefix("Example"),
		weaviate.WithLogger(logger),
	)

	// Example documents
	documents := []interfaces.Document{
		{
			ID:      uuid.New().String(),
			Content: "Artificial intelligence (AI) is intelligence demonstrated by machines, as opposed to natural intelligence displayed by animals including humans.",
			Metadata: map[string]interface{}{
				"source": "wikipedia",
				"topic":  "AI",
				"year":   2023,
			},
		},
		{
			ID:      uuid.New().String(),
			Content: "Machine learning is a subset of artificial intelligence that focuses on the development of algorithms that can learn from and make predictions based on data.",
			Metadata: map[string]interface{}{
				"source": "textbook",
				"topic":  "machine learning",
				"year":   2022,
			},
		},
		{
			ID:      uuid.New().String(),
			Content: "Deep learning is a subset of machine learning that uses neural networks with many layers to analyze various factors of data.",
			Metadata: map[string]interface{}{
				"source": "research paper",
				"topic":  "deep learning",
				"year":   2021,
			},
		},
		{
			ID:      uuid.New().String(),
			Content: "Natural language processing (NLP) is a subfield of linguistics, computer science, and artificial intelligence concerned with the interactions between computers and human language.",
			Metadata: map[string]interface{}{
				"source": "academic journal",
				"topic":  "NLP",
				"year":   2023,
			},
		},
		{
			ID:      uuid.New().String(),
			Content: "Computer vision is an interdisciplinary scientific field that deals with how computers can gain high-level understanding from digital images or videos.",
			Metadata: map[string]interface{}{
				"source": "textbook",
				"topic":  "computer vision",
				"year":   2020,
			},
		},
	}

	// Embedding a single text
	fmt.Println("Embedding a single text...")
	vector, err := embedder.Embed(ctx, "What is artificial intelligence?")
	if err != nil {
		logger.Error(ctx, "Embedding failed", map[string]interface{}{"error": err.Error()})
		return
	}
	fmt.Printf("Generated embedding with %d dimensions\n", len(vector))

	// Storing documents
	fmt.Println("\nStoring documents...")
	err = store.Store(ctx, documents)
	if err != nil {
		logger.Error(ctx, "Failed to store documents", map[string]interface{}{"error": err.Error()})
		return
	}
	fmt.Println("Successfully stored documents")

	// Basic search
	fmt.Println("\nPerforming basic search...")
	results, err := store.Search(ctx, "What is artificial intelligence?", 3)
	if err != nil {
		logger.Error(ctx, "Search failed", map[string]interface{}{"error": err.Error()})
		return
	}
	fmt.Println("Basic search results:")
	printResults(results)

	// Search with filters
	fmt.Println("\nPerforming search with filters...")
	results, err = store.Search(ctx, "What is artificial intelligence?", 3,
		interfaces.WithFilters(map[string]interface{}{
			"source": map[string]interface{}{
				"operator": "equals",
				"value":    "wikipedia",
			},
		}),
	)
	if err != nil {
		logger.Error(ctx, "Search with filters failed", map[string]interface{}{"error": err.Error()})
		return
	}
	fmt.Println("Search results with source=wikipedia filter:")
	printResults(results)

	// Weaviate-compatible filter format
	fmt.Println("\nPerforming search with Weaviate-compatible filters...")
	results, err = store.Search(ctx, "What is artificial intelligence?", 3,
		interfaces.WithFilters(map[string]interface{}{
			"operator":    "Equal",
			"path":        []string{"source"},
			"valueString": "textbook",
		}),
	)
	if err != nil {
		logger.Error(ctx, "Weaviate-compatible filter search failed", map[string]interface{}{"error": err.Error()})
		return
	}
	fmt.Println("Search results with Weaviate-compatible filter (source=textbook):")
	printResults(results)

	// Direct filter format
	fmt.Println("\nPerforming search with direct filter format...")
	results, err = store.Search(ctx, "What is artificial intelligence?", 3,
		interfaces.WithFilters(map[string]interface{}{
			"year": map[string]interface{}{
				"operator": "greaterThan",
				"value":    2021,
			},
		}),
	)
	if err != nil {
		logger.Error(ctx, "Direct filter search failed", map[string]interface{}{"error": err.Error()})
		return
	}
	fmt.Println("Search results with direct filter (year > 2021):")
	printResults(results)

	// Helper function filter format
	fmt.Println("\nPerforming search with helper function filter format...")
	results, err = store.Search(ctx, "What is artificial intelligence?", 3,
		interfaces.WithFilters(map[string]interface{}{
			"topic": map[string]interface{}{
				"operator": "equals",
				"value":    "NLP",
			},
		}),
	)
	if err != nil {
		logger.Error(ctx, "Helper function filter search failed", map[string]interface{}{"error": err.Error()})
		return
	}
	fmt.Println("Search results with helper function filter (topic=NLP):")
	printResults(results)

	// Calculate similarity between two texts
	fmt.Println("\nCalculating similarity between two texts...")
	text1 := "What is artificial intelligence?"
	text2 := "AI is the simulation of human intelligence by machines"

	// Get embeddings for both texts
	vec1, err := embedder.Embed(ctx, text1)
	if err != nil {
		logger.Error(ctx, "Embedding for similarity calculation failed", map[string]interface{}{"error": err.Error()})
		return
	}

	vec2, err := embedder.Embed(ctx, text2)
	if err != nil {
		logger.Error(ctx, "Embedding for similarity calculation failed", map[string]interface{}{"error": err.Error()})
		return
	}

	// Calculate cosine similarity
	similarity, err := calculateCosineSimilarity(vec1, vec2)
	if err != nil {
		logger.Error(ctx, "Similarity calculation failed", map[string]interface{}{"error": err.Error()})
		return
	}
	fmt.Printf("Similarity score: %.4f\n", similarity)

	// Batch embedding
	fmt.Println("\nPerforming batch embedding...")
	texts := []string{
		"What is machine learning?",
		"How does deep learning work?",
		"What are neural networks?",
	}
	vectors, err := embedder.EmbedBatch(ctx, texts)
	if err != nil {
		logger.Error(ctx, "Batch embedding failed", map[string]interface{}{"error": err.Error()})
		return
	}
	fmt.Printf("Generated %d embeddings in batch\n", len(vectors))

	// Clean up
	fmt.Println("\nCleaning up...")
	ids := make([]string, len(documents))
	for i, doc := range documents {
		ids[i] = doc.ID
	}
	err = store.Delete(ctx, ids)
	if err != nil {
		logger.Error(ctx, "Cleanup failed", map[string]interface{}{"error": err.Error()})
		return
	}
	fmt.Println("Successfully cleaned up documents")
}

func printResults(results []interfaces.SearchResult) {
	for i, result := range results {
		fmt.Printf("%d. %s (Score: %.4f)\n", i+1, truncateString(result.Document.Content, 100), result.Score)
		fmt.Printf("   Metadata: %v\n", result.Document.Metadata)
	}
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// calculateCosineSimilarity computes the cosine similarity between two vectors
func calculateCosineSimilarity(vec1, vec2 []float32) (float64, error) {
	if len(vec1) != len(vec2) {
		return 0, fmt.Errorf("vectors must have the same dimensions, got %d and %d", len(vec1), len(vec2))
	}

	var dotProduct float64
	var norm1 float64
	var norm2 float64

	for i := 0; i < len(vec1); i++ {
		dotProduct += float64(vec1[i]) * float64(vec2[i])
		norm1 += float64(vec1[i]) * float64(vec1[i])
		norm2 += float64(vec2[i]) * float64(vec2[i])
	}

	// Avoid division by zero
	if norm1 == 0 || norm2 == 0 {
		return 0, fmt.Errorf("cannot compute similarity for zero vectors")
	}

	return dotProduct / (math.Sqrt(norm1) * math.Sqrt(norm2)), nil
}
