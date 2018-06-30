//go:generate gqlgen -schema ./schema.graphql
package server

import (
	"context"
	"sync"
	"fmt"
	"math/rand"
	"encoding/json"
	"time"

	"github.com/gomodule/redigo/redis"
)

type graphQLServer struct {
	redisConn       redis.Conn
	messageChannels map[string]chan Message
	userChannels    map[string]chan string
	mutex           sync.Mutex
}

func NewGraphQLServer(conn redis.Conn) (*graphQLServer, error) {
	return &graphQLServer{
		redisConn: conn,
	}, nil
}

const messagesQueue = "messages"
const usersQueue = "users"

func (s *graphQLServer) Mutation_postMessage(ctx context.Context, user string, text string) (*Message, error) {
	fmt.Println("call Mutation_postMessage")

	// Create message
	message := &Message{
		ID:   fmt.Sprintf("T%d", rand.Int()),
		User: user,
		Text: text,
		CreatedAt: time.Now().UTC(),
	}

	mj, err := json.Marshal(message)
	if err != nil {
		return nil, err
	}
	s.redisConn.Do("LPUSH", messagesQueue, mj)

	// Notify new message
	s.mutex.Lock()
	for _, ch := range s.messageChannels {
		ch <- *message
	}
	s.mutex.Unlock()
	return message, nil
}

func (s *graphQLServer) Query_messages(ctx context.Context) ([]Message, error) {
	fmt.Println("call Query_messages")
	var messages []Message
	values, err := redis.ByteSlices(s.redisConn.Do("LRANGE", messagesQueue, 0, -1))
	if err != nil {
		return messages, err
	}

	for _, v := range values {
		var m Message
		json.Unmarshal(v, &m)
		messages = append(messages, m)
	}

	return messages, nil
}

func (s *graphQLServer) Query_users(ctx context.Context) ([]string, error) {
	fmt.Println("call Query_users")
	return redis.Strings(s.redisConn.Do("SMEMBERS", usersQueue))
}

func (s *graphQLServer) Subscription_messagePosted(ctx context.Context, user string) (<-chan Message, error) {
	return nil, nil
}

func (s *graphQLServer) Subscription_userJoined(ctx context.Context, user string) (<-chan string, error) {
	return nil, nil
}
