# Saga Pattern Implementation

A Go implementation of the Saga pattern for managing distributed transactions across microservices using Kafka as the message broker.

## Table of Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [Features](#features)
- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Configuration](#configuration)
- [Usage](#usage)
- [Services](#services)
- [Saga Flow](#saga-flow)
- [API Documentation](#api-documentation)
- [Examples](#examples)
- [Testing](#testing)
- [Troubleshooting](#troubleshooting)
- [Contributing](#contributing)

## Overview

This project demonstrates the Saga pattern implementation for managing distributed transactions in a microservices architecture. The system handles order processing workflow with automatic compensation mechanisms when failures occur.

### What is Saga Pattern?

The Saga pattern manages long-running transactions by breaking them into a series of smaller, local transactions. Each local transaction updates data within a single service and publishes an event. If a local transaction fails, the saga executes compensating transactions to undo the changes made by previous transactions.

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│  Order Service  │    │  Stock Service  │    │Purchase Service │    │  Noti Service   │
│                 │    │                 │    │                 │    │                 │
│  REST API       │    │  Stock Check    │    │  Payment        │    │  Notification    │
│  Order Creation │    │  Compensation   │    │  Processing     │    │  Delivery       │
│                 │    │                 │    │                 │    │                 │
└─────────┬───────┘    └─────────┬───────┘    └─────────┬───────┘    └─────────┬───────┘
          │                      │                      │                      │
          └──────────────────────┼──────────────────────┼──────────────────────┘
                                 │                      │
                    ┌────────────┴──────────────────────┴──────────────┐
                    │                Kafka Cluster                     │
                    │                                                  │
                    │  Topics: purchase, checkStock, notification       │
                    │          purchaseFail, checkStockFail,           │
                    │          notificationResult                       │
                    └──────────────────────────────────────────────────┘
```

## Features

- **Choreography-based Saga**: Services coordinate through event publishing
- **Event-driven Architecture**: Uses Kafka for reliable message delivery
- **Compensating Actions**: Automatic rollback on transaction failures
- **REST API**: HTTP endpoints for order management
- **Consumer Groups**: Reliable message processing with offset management
- **JSON Message Format**: Structured event communication
- **Idempotent Operations**: Safe retry mechanisms

## Prerequisites

- Go 1.24.3 or higher
- Apache Kafka (latest)
- Docker (for running Kafka)

## Installation

1. Clone the repository:
```bash
git clone <repository-url>
cd saga-pattern
```

2. Install dependencies:
```bash
go mod download
```

3. Start Kafka using Docker:
```bash
docker-compose up -d
```

## Configuration

The system uses the following Kafka configuration:

```go
// Kafka Broker
Broker: "localhost:9092"

// Topics
Topics:
  - purchase / purchaseFail
  - checkStock / checkStockFail  
  - notification / notificationResult

// Consumer Groups
Groups:
  - stock-service
  - purchase-service
  - notification-service
```

## Usage

### Starting Services

Start each service in separate terminals:

```bash
# Start Order Service (REST API on port 8080)
go run apps/order/main.go

# Start Stock Service
go run apps/stock/main.go

# Start Purchase Service  
go run apps/purchase/purchase.go

# Start Notification Service
go run apps/noti/noti.go
```

### Creating an Order

```bash
$ TBD
```

## Services

### Order Service (`apps/order/main.go`)
- **Port**: 8080
- **Endpoints**: 
  - `POST /order` - Create new order
  - `GET /health` - Health check
- **Function**: Initiates saga by publishing to `checkStock` topic
- **Framework**: Gin HTTP router

### Stock Service (`apps/stock/main.go`)
- **Consumer Group**: `stock-service`
- **Listens**: `checkStock` topic
- **Publishes**: `purchase` (success) or `checkStockFail` (failure)
- **Compensation**: Listens to `purchaseFail` to revert stock

### Purchase Service (`apps/purchase/purchase.go`)
- **Consumer Group**: `purchase-service`  
- **Listens**: `purchase` topic
- **Publishes**: `notification` (success) or `purchaseFail` (failure)
- **Function**: Processes payment transactions

### Notification Service (`apps/noti/noti.go`)
- **Consumer Group**: `notification-service`
- **Listens**: `notification` topic
- **Publishes**: `notificationResult`
- **Function**: Sends order confirmation notifications

## Saga Flow

### Happy Path
```
1. Order Service → checkStock (Order Created)
2. Stock Service → purchase (Stock Available) 
3. Purchase Service → notification (Payment Success)
4. Notification Service → notificationResult (Notification Sent)
```

### Failure Scenarios

#### Stock Check Failure
```
1. Order Service → checkStock
2. Stock Service → checkStockFail (Insufficient Stock)
```

#### Purchase Failure  
```
1. Order Service → checkStock
2. Stock Service → purchase (Stock Reserved)
3. Purchase Service → purchaseFail (Payment Failed)
4. Stock Service → (Compensate: Revert Stock)
```

## API Documentation

### Order Service Endpoints

#### Create Order
```http
POST /order
Content-Type: application/json

{}
```

**Response:**
```json
{
  "order_id": "uuid",
  "status": "created",
  "message": "Order created successfully"
}
```

#### Health Check
```http
GET /health
```

**Response:**
```json
{
  "status": "ok"
}
```

## Examples

### Order Model Structure

```go
type Order struct {
    OrderID    string  `json:"order_id"`
    CustomerID string  `json:"customer_id"`
    ProductID  string  `json:"product_id"`
    Quantity   int     `json:"quantity"`
    Price      float64 `json:"price"`
    Status     string  `json:"status"`
}
```

### Message Publishing

```go
func PublishMessage(topic string, order model.Order) error {
    return utils.PublishKafkaMessage(topic, order)
}
```

### Message Consumption

```go
func ConsumeMessages() {
    reader := kafka.NewReader(kafka.ReaderConfig{
        Brokers:  []string{"localhost:9092"},
        Topic:    client.CheckStock,
        GroupID:  "stock-service",
    })
    defer reader.Close()

    for {
        message, err := reader.ReadMessage(context.Background())
        if err != nil {
            log.Printf("Error reading message: %v", err)
            continue
        }
        
        // Process message...
    }
}
```

## Testing

### Manual Testing

1. Start all services
2. Create an order via REST API
3. Monitor logs to see saga execution
4. Test failure scenarios by stopping services

### Load Testing

```bash
# Create multiple orders
for i in {1..10}; do
  curl -X POST http://localhost:8080/order \
    -H "Content-Type: application/json" \
    -d "{\"customer_id\":\"customer-$i\",\"product_id\":\"product-1\",\"quantity\":1,\"price\":10.00}"
done
```

## Troubleshooting

### Common Issues

#### Kafka Connection Timeout
```
Error: failed to close batch:[7] Request Timed Out
```
**Solutions:**
- Check Kafka is running: `docker ps`
- Verify Kafka port 9092 is accessible
- Increase consumer timeout settings
- Use consumer groups to persist offsets

#### Consumer Group Rebalancing
```
Error: consumer group rebalancing
```
**Solutions:**
- Ensure stable network connection
- Use unique consumer group IDs
- Implement graceful shutdown

#### Messages Not Consuming
**Solutions:**
- Verify topic names match between producer/consumer
- Check consumer group configuration
- Monitor Kafka logs: `docker logs <kafka-container>`

### Debugging Tips

1. **Enable Debug Logging**:
```go
log.SetLevel(log.DebugLevel)
```

2. **Monitor Kafka Topics**:
```bash
docker exec -it <kafka-container> kafka-console-consumer --bootstrap-server localhost:9092 --topic purchase --from-beginning
```

3. **Check Consumer Groups**:
```bash
docker exec -it <kafka-container> kafka-consumer-groups --bootstrap-server localhost:9092 --list
```

## Project Structure

```
saga-pattern/
├── apps/
│   ├── order/main.go          # Order service with REST API
│   ├── stock/main.go          # Stock management service  
│   ├── purchase/purchase.go   # Purchase processing service
│   └── noti/noti.go          # Notification service
├── internal/
│   ├── client/
│   │   ├── client.go         # Kafka client utilities
│   │   └── topic.go          # Topic constants
│   ├── config/config.go      # Configuration structures
│   └── model/order.go        # Order data model
├── utils/
│   ├── kafka.go              # Kafka utility functions
│   └── helper.go             # JSON helpers
├── docker-compose.yaml       # Kafka setup
├── go.mod                    # Go module dependencies
└── README.md                 # This file
```

## Dependencies

- **[segmentio/kafka-go](https://github.com/segmentio/kafka-go)** - Kafka client library
- **[gin-gonic/gin](https://github.com/gin-gonic/gin)** - HTTP web framework  
- **[google/uuid](https://github.com/google/uuid)** - UUID generation

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License.

## Acknowledgments

- [Saga Pattern](https://microservices.io/patterns/data/saga.html) by Chris Richardson
- [Apache Kafka](https://kafka.apache.org/) for reliable messaging
- Go community for excellent libraries