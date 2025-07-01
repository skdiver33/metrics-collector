package main

import (
	"testing"

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
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := Agent{}
			newStorage, err := store.NewMemStorage()
			if err != nil {
				t.Error("error init storage")
			}
			agent.metricStorage = newStorage
			if err != nil {
				t.Errorf("error init agent")
			}
			agent.config = &AgentConfig{serverAddress: "localhost:8080", pollInterval: 2, reportInterval: 5}

			agent.UpdateMetrics()
			if err := agent.SendMetrics(); (err != nil) != tt.wantErr {
				t.Errorf("Agent.SendMetrics() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
