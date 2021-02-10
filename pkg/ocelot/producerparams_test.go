package ocelot

import (
	"net"
	"reflect"
	"testing"

	log "github.com/sirupsen/logrus"
)

func init() {
	// Set Logging
	customFormatter := new(log.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05.0000"
	log.SetFormatter(customFormatter)
	customFormatter.FullTimestamp = true
	log.SetLevel(log.DebugLevel)
}

func TestProducerConfig_newListener(t *testing.T) {
	type fields struct {
		JobChannelBuffer int
		ListenAddr       string
		MaxConnections   int
	}
	tests := []struct {
		name   string
		fields fields
		want   net.Listener
	}{
		{
			name: "T0 - Expected Port",
			fields: fields{
				JobChannelBuffer: 10,
				ListenAddr:       "127.0.0.1:2151",
				MaxConnections:   10,
			},
			want: nil, // See Note on TCP Testing...
		},
		{
			name: "T1 - Port Bind Fail",
			fields: fields{
				JobChannelBuffer: 10,
				ListenAddr:       "127.0.0.1:8000",
				MaxConnections:   10,
			},
			want: nil, // See Note on TCP Testing...
		},
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {

			cfg := &ProducerConfig{
				JobChannelBuffer: tt.fields.JobChannelBuffer,
				ListenAddr:       tt.fields.ListenAddr,
				MaxConnections:   tt.fields.MaxConnections,
			}

			// Create a known connection & Extract Addr Struct
			l, _ := net.Listen("tcp", cfg.ListenAddr)
			wantAddr := l.Addr()
			l.Close()

			// Compare Only Address of Network
			if got := cfg.newListener(); !reflect.DeepEqual(got.Addr(), wantAddr) {
				t.Errorf("ProducerConfig.newListener() = %v, want %v\n", got.Addr(), wantAddr)
			} else {
				log.WithFields(
					log.Fields{"Got": got.Addr(), "Want": wantAddr},
				).Debug()
			}

		})
	}
}
