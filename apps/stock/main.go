package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"saga/internal/client"
	"saga/internal/model"
	"saga/utils"
	"time"

	"github.com/segmentio/kafka-go"
)

type Item struct {
	Name  string
	Stock int32
}

var (
	topics = []client.TOPIC{client.TopicCheckStock, client.TopicPurchaseFail}
	mockDB = []*Item{
		{
			Name:  "watch",
			Stock: 5,
		},
	}
	_clientProducer *kafka.Writer
)

func consume(topic client.TOPIC) {
	// conn, _ := kafka.DialLeader(context.Background(), "tcp", "localhost:9092", string(topic), 0)
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     []string{"localhost:9092"},
		Topic:       string(topic),
		GroupID:     fmt.Sprintf("stock-consumer-group-%s", topic),
		Partition:   0,
		StartOffset: kafka.LastOffset,
	})
	fmt.Println(topic, "consumer ready...")
	defer reader.Close()

	for {
		message, err := reader.FetchMessage(context.Background())
		if err != nil {
			log.Printf("error reading message from %s: %v", topic, err)
			continue
		}
		fmt.Printf("%s message from %s: %s = %s\n", time.Now().Format(time.DateTime), topic, string(message.Key), string(message.Value))
		order := new(model.Order)
		if err := json.Unmarshal(message.Value, order); err != nil {
			log.Printf("error parse payload message from %s: %v", topic, err)
			continue
		}

		switch topic {
		case client.TopicPurchaseFail:
			go handlePurchaseFail(order)
		default:
			handleCheckStock(order)
		}

		if err := reader.CommitMessages(context.Background(), message); err != nil {
			log.Printf("error committing message from %s: %v", topic, err)
		}
	}
}

func handleCheckStock(order *model.Order) {
	var itemDB *Item
	for _, item := range mockDB {
		if item.Name == order.ItemName {
			itemDB = item
			break
		}
	}

	if itemDB != nil && itemDB.Stock > 0 {
		log.Printf("item %s stock is enough, current stock: %d", itemDB.Name, itemDB.Stock)
		// deduct the stock
		itemDB.Stock -= 1
		handlePurchase(order)
	} else {
		handleCheckStockFail(order)
	}
}

func handlePurchase(order *model.Order) {
	msg := utils.CompressToJsonBytes(order)
	err := _clientProducer.WriteMessages(context.Background(), kafka.Message{Key: []byte(order.ID), Topic: string(client.TopicPurchase), Value: msg})
	if err != nil {
		log.Printf("error sending purchase order %s to topic %s: %v", order.ID, client.TopicPurchase, err)
		return
	}
	log.Printf("purchase order %s sent to topic %s", order.ID, client.TopicPurchase)
}

func handleCheckStockFail(order *model.Order) {
	order.Reason = "out of stock"
	order.Success = false
	msg := utils.CompressToJsonBytes(order)
	err := _clientProducer.WriteMessages(context.Background(), kafka.Message{Key: []byte(order.ID), Topic: string(client.TopicCheckStockFail), Value: msg})
	if err != nil {
		log.Printf("error sending check stock fail order %s to topic %s: %v", order.ID, client.TopicCheckStockFail, err)
		return
	}
	log.Printf("check stock fail order %s sent to topic %s", order.ID, client.TopicCheckStockFail)
}

func handlePurchaseFail(order *model.Order) {
	order.Success = false
	order.Reason = "purchase failed"
	msg := utils.CompressToJsonBytes(order)

	// Fallback to check stock fail topic and revert the stock
	var itemDB *Item
	for _, item := range mockDB {
		if item.Name == order.ItemName {
			itemDB = item
			break
		}
	}
	if itemDB != nil {
		itemDB.Stock += 1 // Revert the stock
		log.Printf("reverted stock for item %s, current stock: %d", itemDB.Name, itemDB.Stock)
	}

	err := _clientProducer.WriteMessages(context.Background(), kafka.Message{Key: []byte(order.ID), Topic: string(client.TopicCheckStockFail), Value: msg})
	if err != nil {
		log.Printf("error sending purchase fail order %s to topic %s: %v", order.ID, client.TopicCheckStockFail, err)
		return
	}
	log.Printf("purchase fail order %s sent to topic %s", order.ID, client.TopicCheckStockFail)
}

func main() {
	_clientProducer = &kafka.Writer{
		Addr: kafka.TCP("localhost:9092"),
		// NOTE: When Topic is not defined here, each Message must define it instead.
		Balancer: &kafka.LeastBytes{},
	}

	for _, topic := range topics {
		go consume(topic)
	}

	select {}
}
