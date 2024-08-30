package kafka

type Config struct {
	// A list of Kafka brokers in the form of "host:port"
	Brokers                   []string `mapstructure:"brokers" json:"brokers"`
	OrderTopic                string   `mapstructure:"order_topic" json:"order_topic"`
	OrderStatusTopic          string   `mapstructure:"order_status_topic" json:"order_status_topic"`
	MatchedOrderTopic         string   `mapstructure:"matched_order_topic" json:"matched_order_topic"`
	OrderBookAdminTopic       string   `mapstructure:"order_book_admin_topic" json:"order_book_admin_topic"`
	OrderBookStatusTopic      string   `mapstructure:"order_book_status_topic" json:"order_book_status_topic"`
	OrderBookUpdatesTopic     string   `mapstructure:"order_book_updates_topic" json:"order_book_updates_topic"`
	OrderBookSnapshotTopic    string   `mapstructure:"order_book_snapshots_topic" json:"order_book_snapshots_topic"`
	AssetsRequestTopic        string   `mapstructure:"assets_request_topic" json:"assets_request_topic"`
	AssetSnapshotTopic        string   `mapstructure:"assets_snapshot_topic" json:"assets_snapshot_topic"`
	AssetUpdateTopic          string   `mapstructure:"assets_update_topic" json:"assets_update_topic"`
	BalancesRequestTopic      string   `mapstructure:"balances_request_topic" json:"balances_request_topic"`
	BalanceSnapshotTopic      string   `mapstructure:"balances_snapshot_topic" json:"balances_snapshot_topic"`
	BalanceUpdateTopic        string   `mapstructure:"balances_update_topic" json:"balances_update_topic"`
	UserLedgerRequestsTopic   string   `mapstructure:"user_ledger_requests_topic" json:"user_ledger_requests_topic"`
	UserLedgerSnapshotsTopic  string   `mapstructure:"user_ledger_snapshots_topic" json:"user_ledger_snapshots_topic"`
	UserLedgerUpdatesTopic    string   `mapstructure:"user_ledger_updates_topic" json:"user_ledger_updates_topic"`
	PoolBalanceRequestsTopic  string   `mapstructure:"pool_balance_requests_topic" json:"pool_balance_requests_topic"`
	PoolBalanceSnapshotsTopic string   `mapstructure:"pool_balance_snapshots_topic" json:"pool_balance_snapshots_topic"`
	PoolBalanceUpdatesTopic   string   `mapstructure:"pool_balance_updates_topic" json:"pool_balance_updates_topic"`
}

func DefaultConfig() Config {
	return Config{
		Brokers:                   nil,
		OrderTopic:                DefaultOrderTopic,
		OrderStatusTopic:          DefaultOrderStatusTopic,
		MatchedOrderTopic:         DefaultMatchedOrderTopic,
		OrderBookAdminTopic:       DefaultOrderBookAdminTopic,
		OrderBookStatusTopic:      DefaultOrderBookStatusTopic,
		OrderBookUpdatesTopic:     DefaultOrderBookUpdatesTopic,
		OrderBookSnapshotTopic:    DefaultOrderBookSnapshotTopic,
		AssetsRequestTopic:        DefaultAssetRequestsTopic,
		AssetSnapshotTopic:        DefaultAssetSnapshotTopic,
		AssetUpdateTopic:          DefaultAssetUpdatesTopic,
		BalancesRequestTopic:      DefaultBalanceRequestsTopic,
		BalanceSnapshotTopic:      DefaultBalanceSnapshotTopic,
		BalanceUpdateTopic:        DefaultBalanceUpdateTopic,
		UserLedgerRequestsTopic:   DefaultUserLedgerRequestsTopic,
		UserLedgerSnapshotsTopic:  DefaultUserLedgerSnapshotsTopic,
		UserLedgerUpdatesTopic:    DefaultUserLedgerUpdatesTopic,
		PoolBalanceRequestsTopic:  DefaultPoolBalanceRequestsTopic,
		PoolBalanceSnapshotsTopic: DefaultPoolBalanceSnapshotsTopic,
		PoolBalanceUpdatesTopic:   DefaultPoolBalanceUpdatesTopic,
	}
}
