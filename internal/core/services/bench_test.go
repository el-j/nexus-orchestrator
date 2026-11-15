package services_test

import (
	"fmt"
	"testing"

	"nexus-orchestrator/internal/core/ports"
	"nexus-orchestrator/internal/core/services"
)

// BenchmarkFindForModel is the router's lookup with fifty providers registered; the model is served by
// the last one, so the whole list is searched.
func BenchmarkFindForModel(b *testing.B) {
	var clients []ports.LLMClient
	for i := 0; i < 50; i++ {
		clients = append(clients, &scriptLLM{name: fmt.Sprintf("p%02d", i), alive: true, model: fmt.Sprintf("m%02d", i), models: []string{fmt.Sprintf("m%02d", i)}})
	}
	d := services.NewDiscoveryService(clients...)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := d.FindForModel("m49", ""); err != nil {
			b.Fatal(err)
		}
	}
}
