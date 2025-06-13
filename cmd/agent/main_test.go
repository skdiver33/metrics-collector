package main

import (
	"testing"

	"github.com/skdiver33/metrics-collector/internal/store"
)

func TestAgent_SendMetrics(t *testing.T) {
	type fields struct {
		metricStorage store.MemStorage
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name:    "positive test",
			fields:  fields{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := &Agent{
				metricStorage: tt.fields.metricStorage,
				config:        AgentConfig{serverAddress: "localhost:8080", pollInterval: 2, reportInterval: 10},
			}
			agent.metricStorage.InitializeStorage()
			agent.UpdateMetrics()
			if err := agent.SendMetrics(); (err != nil) != tt.wantErr {
				t.Errorf("Agent.SendMetrics() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
