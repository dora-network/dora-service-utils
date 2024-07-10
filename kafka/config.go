package kafka

type Config struct {
	// A list of Kafka brokers in the form of "host:port"
	Brokers                []string `mapstructure:"brokers" json:"brokers"`
	OrderTopic             string   `mapstructure:"order_topic" json:"order_topic"`
	OrderStatusTopic       string   `mapstructure:"order_status_topic" json:"order_status_topic"`
	MatchedOrderTopic      string   `mapstructure:"matched_order_topic" json:"matched_order_topic"`
	OrderBookAdminTopic    string   `mapstructure:"order_book_admin_topic" json:"order_book_admin_topic"`
	OrderBookUpdatesTopic  string   `mapstructure:"order_book_updates_topic" json:"order_book_updates_topic"`
	OrderBookSnapshotTopic string   `mapstructure:"order_book_snapshot_topic" json:"order_book_snapshot_topic"`
}
