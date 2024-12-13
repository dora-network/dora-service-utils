package kafka

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"os"
	"strings"
	"testing"
)

func TestConsumerGroup(t *testing.T) {
	type args struct {
		hostname  string
		component string
	}

	mac, err := getMacAddr()
	require.NoError(t, err)

	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "no hostname",
			args: args{
				hostname:  "",
				component: "test",
			},
			want: fmt.Sprintf("%s-%s", strings.Join(mac, ":"), "test"),
		},
		{
			name: "with hostname",
			args: args{
				hostname:  "hostname",
				component: "test",
			},
			want: "hostname-test",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(tt *testing.T) {
			if tc.args.hostname == "" {
				require.NoError(tt, os.Unsetenv("HOSTNAME"))
			} else {
				require.NoError(tt, os.Setenv("HOSTNAME", tc.args.hostname))
			}
			got := ConsumerGroup(tc.args.component)
			require.Equal(tt, tc.want, got)
		})
	}
}
