package client

type TOPIC string

const (
	TopicPurchase           TOPIC = "purchase"
	TopicPurchaseFail       TOPIC = "purchaseFail"
	TopicCheckStock         TOPIC = "checkStock"
	TopicCheckStockFail     TOPIC = "checkStockFail"
	TopicNotification       TOPIC = "notification"
	TopicNotificationResult TOPIC = "notificationResult"
)
