package main

import (
	"context"
	"encoding/json"
	"fmt"
	"saga/internal/client"
	"saga/internal/model"
	"saga/utils"

	"github.com/segmentio/kafka-go"
)

type ThirdPartyService struct {
	// This struct can hold any necessary fields for the third-party service
	Success bool
}

var (
	_clientProducer    *kafka.Writer
	_thirdPartyService *ThirdPartyService = &ThirdPartyService{
		Success: true, // Simulate a successful third-party service call
	}
)

func consume() {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     []string{"localhost:9092"},
		Topic:       string(client.TopicPurchase),
		GroupID:     "purchase-consumer-group",
		Partition:   0,
		StartOffset: kafka.LastOffset,
	})
	defer reader.Close()
	fmt.Println(client.TopicPurchase, "consumer ready...")

	for {
		message, err := reader.ReadMessage(context.Background())
		if err != nil {
			fmt.Printf("error reading message from %s: %v\n", client.TopicPurchase, err)
			break
		}
		fmt.Printf("%s message from %s: %s = %s\n", message.Time, client.TopicPurchase, string(message.Key), string(message.Value))

		// Process the message here
		order := new(model.Order)
		if err := json.Unmarshal(message.Value, order); err != nil {
			fmt.Printf("error unmarshalling message: %v\n", err)
			continue
		}
		handlePurchase(order)
	}
}

func handlePurchase(order *model.Order) {
	// Process the purchase order
	fmt.Printf("Processing purchase order: %s\n", order.ID)

	// Here you would typically interact with a database or another service
	if _thirdPartyService.Success {
		handleNotification(order)
	} else {
		handlePurchaseFail(order)
	}
}

func handleNotification(order *model.Order) {
	// Simulate sending a notification
	fmt.Printf("Notification sent for order %s.\n", order.ID)

	msg := utils.CompressToJsonBytes(order)
	err := _clientProducer.WriteMessages(context.Background(), kafka.Message{
		Value: msg,
		Topic: string(client.TopicNotification),
	})
	if err != nil {
		fmt.Printf("error sending notification for order %s: %v\n", order.ID, err)
		return
	}
	fmt.Printf("Notification for order %s sent successfully.\n", order.ID)
}

func handlePurchaseFail(order *model.Order) {
	fmt.Printf("Handling purchase fail for order %s...\n", order.ID)
	order.Success = false // Mark the order as failed
	order.Reason = "third-party service failure"

	err := _clientProducer.WriteMessages(context.Background(), kafka.Message{
		Value: utils.CompressToJsonBytes(order),
		Topic: string(client.TopicPurchaseFail),
	})
	if err != nil {
		fmt.Printf("error sending purchase fail message for order %s: %v\n", order.ID, err)
		return
	}

	fmt.Printf("purchase fail for order %s completed.\n", order.ID)
}

func main() {
	// Initialize the Kafka producer
	// var err error
	_clientProducer = kafka.NewWriter(kafka.WriterConfig{
		Brokers:  []string{"localhost:9092"},
		Balancer: &kafka.LeastBytes{},
	})
	defer func() {
		if err := _clientProducer.Close(); err != nil {
			fmt.Printf("failed to close Kafka producer: %v\n", err)
		}
	}()

	go consume()

	// Keep the main function running
	select {}
}
