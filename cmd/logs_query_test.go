package cmd

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestBuildQuery(t *testing.T) {
	tests := []struct {
		name    string
		flags   map[string]string
		args    []string
		want    string
	}{
		{
			name: "no flags no args",
			want: "*",
		},
		{
			name:  "service only",
			flags: map[string]string{"service": "payment"},
			want:  "service:payment",
		},
		{
			name:  "env only",
			flags: map[string]string{"env": "prod"},
			want:  "env:prod",
		},
		{
			name:  "host only",
			flags: map[string]string{"host": "web-1"},
			want:  "host:web-1",
		},
		{
			name:  "status only",
			flags: map[string]string{"status": "error"},
			want:  "status:error",
		},
		{
			name:  "all flags",
			flags: map[string]string{"service": "payment", "env": "prod", "host": "web-1", "status": "error"},
			want:  "service:payment env:prod host:web-1 status:error",
		},
		{
			name: "query arg only",
			args: []string{"@duration:>5s"},
			want: "@duration:>5s",
		},
		{
			name:  "flags combined with query arg",
			flags: map[string]string{"service": "payment", "env": "prod"},
			args:  []string{"@duration:>5s"},
			want:  "service:payment env:prod @duration:>5s",
		},
		{
			name:  "service flag with complex query",
			flags: map[string]string{"service": "web-store"},
			args:  []string{"status:error OR status:warn"},
			want:  "service:web-store status:error OR status:warn",
		},
		{
			name: "empty query arg treated as absent",
			args: []string{""},
			want: "*",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			addLogFilterFlags(cmd)

			for k, v := range tt.flags {
				cmd.Flags().Set(k, v)
			}

			got := buildQuery(cmd, tt.args)
			assert.Equal(t, tt.want, got)
		})
	}
}
