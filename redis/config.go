package redis

type Config struct {
	Address         []string `mapstructure:"address"  json:"address,omitempty"`
	Password        string   `mapstructure:"password" json:"password,omitempty"`
	DB              int      `mapstructure:"db" json:"db,omitempty"`
	Protocol        int      `mapstructure:"protocol" json:"protocol,omitempty"`
	DisableIdentity bool     `mapstructure:"disable_identity" json:"disable_identity,omitempty"`
	UseCluster      bool     `mapstructure:"use_cluster" json:"use_cluster,omitempty"`
}
