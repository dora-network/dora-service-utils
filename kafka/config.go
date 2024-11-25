package kafka

type Config struct {
	// A list of Kafka brokers in the form of "host:port"
	Brokers                     []string `mapstructure:"brokers" json:"brokers"`
	OrderTopic                  string   `mapstructure:"order_topic" json:"order_topic"`
	OrderStatusTopic            string   `mapstructure:"order_status_topic" json:"order_status_topic"`
	MatchedOrderTopic           string   `mapstructure:"matched_order_topic" json:"matched_order_topic"`
	OrderBookAdminTopic         string   `mapstructure:"order_book_admin_topic" json:"order_book_admin_topic"`
	OrderBookStatusTopic        string   `mapstructure:"order_book_status_topic" json:"order_book_status_topic"`
	OrderBookUpdatesTopic       string   `mapstructure:"order_book_updates_topic" json:"order_book_updates_topic"`
	OrderBookSnapshotTopic      string   `mapstructure:"order_book_snapshots_topic" json:"order_book_snapshots_topic"`
	OrderBookRequestsTopic      string   `mapstructure:"order_book_requests_topic" json:"order_book_requests_topic"`
	AssetsRequestTopic          string   `mapstructure:"assets_request_topic" json:"assets_request_topic"`
	AssetSnapshotTopic          string   `mapstructure:"assets_snapshot_topic" json:"assets_snapshot_topic"`
	AssetUpdateTopic            string   `mapstructure:"assets_update_topic" json:"assets_update_topic"`
	UserLedgerRequestsTopic     string   `mapstructure:"user_ledger_requests_topic" json:"user_ledger_requests_topic"`
	UserLedgerSnapshotsTopic    string   `mapstructure:"user_ledger_snapshots_topic" json:"user_ledger_snapshots_topic"`
	UserLedgerUpdatesTopic      string   `mapstructure:"user_ledger_updates_topic" json:"user_ledger_updates_topic"`
	LeverageRequestsTopic       string   `mapstructure:"leverage_requests_topic" json:"leverage_requests_topic"`
	LeverageSnapshotsTopic      string   `mapstructure:"leverage_snapshots_topic" json:"leverage_snapshots_topic"`
	LeverageUpdatesTopic        string   `mapstructure:"leverage_updates_topic" json:"leverage_updates_topic"`
	PoolLiquidityRequestsTopic  string   `mapstructure:"pool_liquidity_requests_topic" json:"pool_liquidity_requests_topic"`
	PoolLiquiditySnapshotsTopic string   `mapstructure:"pool_liquidity_snapshots_topic" json:"pool_liquidity_snapshots_topic"`
	PoolLiquidityUpdatesTopic   string   `mapstructure:"pool_liquidity_updates_topic" json:"pool_liquidity_updates_topic"`
	TransactionsTopic           string   `mapstructure:"transactions_topic" json:"transactions_topic"`
	AddTransactionRequestTopic  string   `mapstructure:"add_transactions_request_topic" json:"add_transactions_request_topic"`
	AddTransactionResponseTopic string   `mapstructure:"add_transactions_response_topic" json:"add_transactions_response_topic"`
	TradesRequestTopic          string   `mapstructure:"trades_request_topic" json:"trades_request_topic"`
	TradesResponseTopic         string   `mapstructure:"trades_response_topic" json:"trades_response_topic"`
	TradesUpdateTopic           string   `mapstructure:"trades_update_topic" json:"trades_update_topic"`
	Authentication              Auth     `mapstructure:"authentication" json:"authentication"`
}

func DefaultConfig() Config {
	return Config{
		Brokers:                     nil,
		OrderTopic:                  DefaultOrderTopic,
		OrderStatusTopic:            DefaultOrderStatusTopic,
		MatchedOrderTopic:           DefaultMatchedOrderTopic,
		OrderBookAdminTopic:         DefaultOrderBookAdminTopic,
		OrderBookStatusTopic:        DefaultOrderBookStatusTopic,
		OrderBookUpdatesTopic:       DefaultOrderBookUpdatesTopic,
		OrderBookSnapshotTopic:      DefaultOrderBookSnapshotTopic,
		OrderBookRequestsTopic:      DefaultOrderBookRequestsTopic,
		AssetsRequestTopic:          DefaultAssetRequestsTopic,
		AssetSnapshotTopic:          DefaultAssetSnapshotTopic,
		AssetUpdateTopic:            DefaultAssetUpdatesTopic,
		UserLedgerRequestsTopic:     DefaultUserLedgerRequestsTopic,
		UserLedgerSnapshotsTopic:    DefaultUserLedgerSnapshotsTopic,
		UserLedgerUpdatesTopic:      DefaultUserLedgerUpdatesTopic,
		LeverageRequestsTopic:       DefaultLeverageRequestsTopic,
		LeverageSnapshotsTopic:      DefaultLeverageSnapshotsTopic,
		LeverageUpdatesTopic:        DefaultLeverageUpdatesTopic,
		PoolLiquidityRequestsTopic:  DefaultPoolLiquidityRequestsTopic,
		PoolLiquiditySnapshotsTopic: DefaultPoolLiquiditySnapshotsTopic,
		PoolLiquidityUpdatesTopic:   DefaultPoolLiquidityUpdatesTopic,
		TransactionsTopic:           DefaultTransactionsTopic,
		AddTransactionRequestTopic:  DefaultAddTransactionsRequestTopic,
		AddTransactionResponseTopic: DefaultAddTransactionsResponseTopic,
		TradesRequestTopic:          DefaultTradesRequestTopic,
		TradesResponseTopic:         DefaultTradesResponseTopic,
		TradesUpdateTopic:           DefaultTradesUpdateTopic,
	}
}
