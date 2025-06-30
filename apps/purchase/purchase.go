package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"saga/internal/client"
	"saga/internal/model"
	"saga/utils"

	"github.com/avast/retry-go"
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
		message, err := reader.FetchMessage(context.Background())
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
		err = handlePurchase(order)
		if err != nil {
			fmt.Printf("error handling purchase for order %s: %v\n", order.ID, err)
			continue
		} else {
			fmt.Printf("purchase for order %s handled successfully.\n", order.ID)
		}

		// Commit the message after processing
		if err := reader.CommitMessages(context.Background(), message); err != nil {
			log.Printf("error committing message from %s: %v", client.TopicPurchase, err)
		}
	}
}

func handlePurchase(order *model.Order) error {
	// Process the purchase order
	fmt.Printf("Processing purchase order: %s\n", order.ID)

	// Here you would typically interact with a database or another service
	if _thirdPartyService.Success {
		return handleNotification(order)
	}

	return handlePurchaseFail(order)
}

func handleNotification(order *model.Order) error {
	err := writeMessagesWithRetry(context.Background(), kafka.Message{
		Key:   []byte(order.ID),
		Topic: string(client.TopicNotification),
		Value: utils.CompressToJsonBytes(order),
	})
	if err != nil {
		fmt.Printf("error sending notification for order %s: %v\n", order.ID, err)
		return err
	}

	fmt.Printf("Notification for order %s sent successfully.\n", order.ID)
	return nil
}

func handlePurchaseFail(order *model.Order) error {
	order.Success = false // Mark the order as failed
	order.Reason = "third-party service failure"

	err := writeMessagesWithRetry(context.Background(), kafka.Message{
		Key:   []byte(order.ID),
		Topic: string(client.TopicPurchaseFail),
		Value: utils.CompressToJsonBytes(order),
	})
	if err != nil {
		fmt.Printf("error sending purchase fail message for order %s: %v\n", order.ID, err)
		return err
	}

	fmt.Printf("purchase fail for order %s completed.\n", order.ID)
	return nil
}

func writeMessagesWithRetry(ctx context.Context, messages ...kafka.Message) error {
	err := retry.Do(func() error {
		return _clientProducer.WriteMessages(ctx, messages...)
	}, retry.Attempts(3), retry.LastErrorOnly(true))
	return err
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
