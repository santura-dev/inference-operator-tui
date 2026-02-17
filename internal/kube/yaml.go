package kube

import (
	"fmt"
	"strings"

	"github.com/santura-dev/inference-operator-tui/internal/config"
)

// GenerateYAML renders the LocalInferenceService manifest for a config.
func GenerateYAML(c config.DeploymentConfig) string {
	var b strings.Builder
	fmt.Fprintf(&b, `apiVersion: serving.local-ome.com/v1
kind: LocalInferenceService
metadata:
  name: deployed-model
  namespace: default
spec:
  runtime: %s
  model:
    uri: "%s"
    name: "Deployed Model"
  settings:
    batchSize: %d
    precision: %s
    maxTokens: %d
  scaling:
    replicas: %d`, c.Runtime, c.Model, c.BatchSize, c.Precision, c.MaxTokens, c.Replicas)
	if c.LoRA != "" {
		fmt.Fprintf(&b, "\n  lora:\n    loraPath: %q", c.LoRA)
	}
	return b.String()
}
