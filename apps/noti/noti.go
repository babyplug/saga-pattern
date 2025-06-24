package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"saga/internal/client"
	"saga/internal/model"
	"saga/utils"

	"github.com/segmentio/kafka-go"
)

var (
	_clientProducer *kafka.Conn
	_clientConsumer *kafka.Reader
	TopicConsumer   = client.TopicNotification
	TopicProducer   = client.TopicNotificationResult
)

func consume() {
	fmt.Println(TopicConsumer, "consumer ready...")
	for {
		message, err := _clientConsumer.ReadMessage(context.Background())
		if err != nil {
			log.Printf("error reading message from %s: %v\n", TopicConsumer, err)
			break
		}
		fmt.Printf("%s message from %s: %s = %s\n", message.Time, TopicConsumer, string(message.Key), string(message.Value))

		// Process the message here
		order := new(model.Order)
		if err := json.Unmarshal(message.Value, order); err != nil {
			fmt.Printf("error unmarshalling message: %v\n", err)
			continue
		}
		handleNotification(order)
	}
}

func handleNotification(order *model.Order) {
	order.Reason = "Order processed successfully"

	msg := utils.CompressToJsonBytes(order)
	_, err := _clientProducer.Write(msg)
	if err != nil {
		log.Printf("error sending notification  %s: %v\n", order.ID, err)
		return
	}
}

func main() {
	// Initialize the Kafka producer
	var err error
	_clientProducer, err = kafka.DialLeader(context.Background(), "tcp", "localhost:9092", string(TopicProducer), 0)
	defer func() {
		err := _clientProducer.Close()
		if err != nil {
			log.Fatalf("failed to close Kafka producer: %v", err)
		}
	}()
	if err != nil {
		log.Fatalf("failed to connect to Kafka producer: %v", err)
		return
	}

	_clientConsumer = kafka.NewReader(kafka.ReaderConfig{
		Brokers:     []string{"localhost:9092"},
		Topic:       string(TopicConsumer),
		GroupID:     "notification-group",
		Partition:   0,
		StartOffset: kafka.LastOffset,
	})
	defer func() {
		err := _clientConsumer.Close()
		if err != nil {
			log.Fatalf("failed to close Kafka consumer: %v", err)
		}
	}()
	if err != nil {
		log.Fatalf("failed to connect to Kafka consumer: %v", err)
		return
	}
	go consume()

	// Keep the main function running
	select {}
}
