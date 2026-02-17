package kube

import (
	"strings"
	"testing"

	"github.com/santura-dev/inference-operator-tui/internal/config"
)

func TestGenerateYAML(t *testing.T) {
	y := GenerateYAML(config.DeploymentConfig{
		Model: "m", Runtime: "vllm", Precision: "fp16",
		MaxTokens: 7, Replicas: 2, BatchSize: 3, LoRA: "/path",
	})
	for _, want := range []string{
		`runtime: vllm`, `uri: "m"`, `batchSize: 3`, `precision: fp16`,
		`maxTokens: 7`, `replicas: 2`, `loraPath: "/path"`,
	} {
		if !strings.Contains(y, want) {
			t.Errorf("yaml missing %q:\n%s", want, y)
		}
	}
}

func TestGenerateYAMLNoLora(t *testing.T) {
	y := GenerateYAML(config.DeploymentConfig{Model: "m", Runtime: "sglang", Precision: "fp16", MaxTokens: 1, Replicas: 1, BatchSize: 1})
	if strings.Contains(y, "lora") {
		t.Errorf("lora section present for empty path:\n%s", y)
	}
}
