package config

import "flag"

// DeploymentConfig describes a LocalInferenceService to deploy.
type DeploymentConfig struct {
	Model     string
	Runtime   string
	Precision string
	MaxTokens int
	Replicas  int
	LoRA      string
	BatchSize int
}

// Options returns the settable fields in list order.
func (c *DeploymentConfig) Fields() []string {
	return []string{"Model", "Runtime", "Precision", "Max Tokens", "Replicas", "LoRA Path", "Batch Size"}
}

// Load parses CLI flags that pre-fill the new-deployment form.
func Load() *DeploymentConfig {
	cfg := &DeploymentConfig{
		Model:     "facebook/opt-125m",
		Runtime:   "sglang",
		Precision: "fp16",
		MaxTokens: 100,
		Replicas:  1,
		BatchSize: 1,
	}
	flag.StringVar(&cfg.Model, "model", cfg.Model, "pre-select model URI")
	flag.StringVar(&cfg.Runtime, "runtime", cfg.Runtime, "pre-select runtime (sglang or vllm)")
	flag.Parse()
	return cfg
}
