package utils

import (
	"context"
	"saga/internal/config"

	"github.com/segmentio/kafka-go"
)

func KafkaConn(cfg config.KafkaConnCfg) *kafka.Conn {
	conn, err := kafka.DialLeader(context.Background(), "tcp", cfg.Url, cfg.Topic, 0)
	if err != nil {
		panic(err.Error())
	}
	return conn
}

func IsTopicAlreadyExists(conn *kafka.Conn, topic string) bool {
	partitions, err := conn.ReadPartitions()
	if err != nil {
		panic(err.Error())
	}

	for _, p := range partitions {
		if p.Topic == topic {
			return true
		}
	}
	return false
}
