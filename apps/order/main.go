package main

import (
	"context"
	"fmt"
	"log"
	"saga/internal/client"
	"saga/internal/model"
	"saga/utils"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

var (
	_clientProducer *kafka.Conn
	_topics         = []client.TOPIC{
		client.TopicCheckStockFail,
		client.TopicNotificationResult,
	}
)

var checkStock = func(ctx *gin.Context) {
	id := uuid.NewString()

	order := model.Order{
		ID:       id,
		ItemName: "watch",
		Success:  true,
	}

	msg := utils.CompressToJsonBytes(&order)
	// Set timeout
	_clientProducer.SetWriteDeadline(time.Now().Add(10 * time.Second))
	_, err := _clientProducer.WriteMessages(kafka.Message{
		Topic: string(client.TopicCheckStock),
		Key:   []byte(id),
		Value: msg,
	})
	if err != nil {
		log.Fatal("failed to write messages:", err)
	}

	ctx.JSON(201, gin.H{
		"message":  "Send order done",
		"order_id": id,
		"status":   "created",
	})
}

func consume(topic client.TOPIC) {
	// conn, _ := kafka.DialLeader(context.Background(), "tcp", "localhost:9092", string(topic), 0)
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     []string{"localhost:9092"},
		Topic:       string(topic),
		GroupID:     "order-group-" + string(topic),
		Partition:   0,
		StartOffset: kafka.LastOffset,
	})
	for {
		msg, err := reader.FetchMessage(context.Background())
		if err != nil {
			log.Printf("failed to read message: %v", err)
			continue
		}
		fmt.Printf("%s message from %s: %s = %s\n", time.Now().Format(time.DateTime), topic, string(msg.Key), string(msg.Value))
		err = reader.CommitMessages(context.Background(), msg)
		if err != nil {
			log.Printf("Error committing message topic %s: %v", topic, err)
		}
	}
}

func main() {
	var err error
	_clientProducer, err = kafka.DialLeader(context.Background(), "tcp", "localhost:9092", string(client.TopicCheckStock), 0)
	if err != nil {
		log.Fatalf("subscribe to topic %s failed: %v", client.TopicCheckStock, err)
	}

	for _, topic := range _topics {
		go consume(topic)
	}

	router := gin.Default()
	router.POST("/order", checkStock)
	router.GET("/health", func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{
			"status": "ok",
		})
	})

	router.Run() // listen and serve on 0.0.0.0:8080
}
