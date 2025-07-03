package main

import (
	"testing"

	"github.com/skdiver33/metrics-collector/internal/agent"
	"github.com/skdiver33/metrics-collector/internal/store"
)

func TestAgent_SendMetrics(t *testing.T) {
	// type fields struct {
	// 	metricStorage store.MemStorage
	// }
	tests := []struct {
		name string
		//fields  fields
		wantErr bool
	}{
		{
			name: "positive test",
			//fields:  fields{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newStorage, err := store.NewMemStorage()
			if err != nil {
				t.Error("error init storage")
			}
			agent, err := agent.NewAgent(newStorage)
			if err != nil {
				t.Error("error inicreatet agent")
			}

			agent.UpdateMetrics()
			if err := agent.UpdateMetrics(); (err != nil) != tt.wantErr {
				t.Errorf("Agent.SendMetrics() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
