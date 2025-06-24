package client

import (
	"context"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

const (
	batchSize = int(10e6)
)

func New(ctx context.Context, topic TOPIC, partitions ...int) (*kafka.Conn, error) {
	partition := 0
	if len(partitions) > 0 {
		partition = partitions[0]
	}
	conn, err := kafka.DialLeader(ctx, "tcp", "localhost:9092", string(topic), partition)
	if err != nil {
		log.Fatal("failed to dial leader:", err)
	}

	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))

	return conn, err
}

func NewReader(topic TOPIC, groupID string) *kafka.Reader {
	return kafka.NewReader(kafka.ReaderConfig{
		Brokers:     []string{"localhost:9092"},
		GroupID:     groupID,
		Topic:       string(topic),
		// Partition:   0,
		StartOffset: kafka.LastOffset,
		MinBytes:    10e3,      // 10KB
		MaxBytes:    batchSize, // 10MB
	})
}
