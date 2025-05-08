package redis

type ClientType int

const (
	ClientTypeRegular ClientType = iota
	ClientTypeCluster
	ClientTypeFailover
)

func (c ClientType) String() string {
	switch c {
	case ClientTypeRegular:
		return "regular"
	case ClientTypeCluster:
		return "cluster"
	case ClientTypeFailover:
		return "failover"
	default:
		return "unspecified"
	}
}

func ClientTypeFromString(clientType string) ClientType {
	switch clientType {
	case "cluster":
		return ClientTypeCluster
	case "failover":
		return ClientTypeFailover
	default:
		return ClientTypeRegular
	}
}

type Config struct {
	Address         []string   `mapstructure:"address"  json:"address,omitempty"`
	Username        string     `mapstructure:"username" json:"username,omitempty"`
	Password        string     `mapstructure:"password" json:"password,omitempty"`
	DB              int        `mapstructure:"db" json:"db,omitempty"`
	Protocol        int        `mapstructure:"protocol" json:"protocol,omitempty"`
	DisableIdentity bool       `mapstructure:"disable_identity" json:"disable_identity,omitempty"`
	ClientType      ClientType `mapstructure:"client_type" json:"client_type,omitempty"`
	MasterName      string     `mapstructure:"master_name"  json:"master_name,omitempty"`
}
