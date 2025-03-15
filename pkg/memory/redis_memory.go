package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"

	"github.com/Ingenimax/agent-sdk-go/pkg/interfaces"
)

// RedisMemory implements a memory that stores messages in Redis
type RedisMemory struct {
	client *redis.Client
	ttl    time.Duration
}

// RedisOption represents an option for configuring the Redis memory
type RedisOption func(*RedisMemory)

// WithTTL sets the time-to-live for messages in Redis
func WithTTL(ttl time.Duration) RedisOption {
	return func(r *RedisMemory) {
		r.ttl = ttl
	}
}

// NewRedisMemory creates a new Redis memory
func NewRedisMemory(client *redis.Client, options ...RedisOption) *RedisMemory {
	memory := &RedisMemory{
		client: client,
		ttl:    24 * time.Hour, // Default TTL
	}

	for _, option := range options {
		option(memory)
	}

	return memory
}

// AddMessage adds a message to the memory
func (r *RedisMemory) AddMessage(ctx context.Context, message interfaces.Message) error {
	// Get conversation ID from context
	conversationID, err := getConversationID(ctx)
	if err != nil {
		return err
	}

	// Add timestamp if not present
	if _, ok := message.Metadata["timestamp"]; !ok {
		if message.Metadata == nil {
			message.Metadata = make(map[string]interface{})
		}
		message.Metadata["timestamp"] = time.Now().UnixNano()
	}

	// Serialize message
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Get next message index
	key := fmt.Sprintf("%s:count", conversationID)
	index, err := r.client.Incr(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("failed to get next message index: %w", err)
	}

	// Store message
	messageKey := fmt.Sprintf("%s:msg:%d", conversationID, index)
	if err := r.client.Set(ctx, messageKey, data, r.ttl).Err(); err != nil {
		return fmt.Errorf("failed to store message: %w", err)
	}

	// Update TTL for count key
	if err := r.client.Expire(ctx, key, r.ttl).Err(); err != nil {
		return fmt.Errorf("failed to set TTL for count key: %w", err)
	}

	return nil
}

// GetMessages retrieves messages from the memory
func (r *RedisMemory) GetMessages(ctx context.Context, options ...interfaces.GetMessagesOption) ([]interfaces.Message, error) {
	// Get conversation ID from context
	conversationID, err := getConversationID(ctx)
	if err != nil {
		return nil, err
	}

	// Parse options
	opts := &interfaces.GetMessagesOptions{}
	for _, option := range options {
		option(opts)
	}

	// Get message count
	key := fmt.Sprintf("%s:count", conversationID)
	count, err := r.client.Get(ctx, key).Int64()
	if err == redis.Nil {
		// No messages
		return []interfaces.Message{}, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to get message count: %w", err)
	}

	// Calculate range
	start := int64(1)
	end := count
	if opts.Limit > 0 && int64(opts.Limit) < count {
		start = count - int64(opts.Limit) + 1
	}

	// Get messages
	var messages []interfaces.Message
	for i := start; i <= end; i++ {
		messageKey := fmt.Sprintf("%s:msg:%d", conversationID, i)
		data, err := r.client.Get(ctx, messageKey).Bytes()
		if err == redis.Nil {
			// Message expired or deleted
			continue
		} else if err != nil {
			return nil, fmt.Errorf("failed to get message %d: %w", i, err)
		}

		// Deserialize message
		var message interfaces.Message
		if err := json.Unmarshal(data, &message); err != nil {
			return nil, fmt.Errorf("failed to unmarshal message %d: %w", i, err)
		}

		// Filter by role if specified
		if len(opts.Roles) > 0 {
			found := false
			for _, role := range opts.Roles {
				if message.Role == role {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		messages = append(messages, message)
	}

	return messages, nil
}

// Clear clears the memory
func (r *RedisMemory) Clear(ctx context.Context) error {
	// Get conversation ID from context
	conversationID, err := getConversationID(ctx)
	if err != nil {
		return err
	}

	// Get message count
	key := fmt.Sprintf("%s:count", conversationID)
	count, err := r.client.Get(ctx, key).Int64()
	if err == redis.Nil {
		// No messages
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to get message count: %w", err)
	}

	// Delete messages
	var keys []string
	keys = append(keys, key)
	for i := int64(1); i <= count; i++ {
		keys = append(keys, fmt.Sprintf("%s:msg:%d", conversationID, i))
	}

	if err := r.client.Del(ctx, keys...).Err(); err != nil {
		return fmt.Errorf("failed to delete messages: %w", err)
	}

	return nil
}
