# inference-operator-tui

![Go](https://img.shields.io/badge/go-%2300ADD8.svg?style=flat&logo=go&logoColor=white) ![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg) ![Kubernetes](https://img.shields.io/badge/kubernetes-compatible-blue?logo=kubernetes)

Terminal UI for local-inference-operator resources. CRDs, pod status, events, logs, GPU utilization in one view.

## The problem

`kubectl get localinferenceservices` shows what models are declared. `kubectl get pods` shows pod status. `kubectl describe` shows events. `kubectl logs` shows server output. `nvidia-smi` shows GPU utilization. None of these talk to each other. When a model fails to load, you are running four commands in four terminals to figure out why.

## The idea

One TUI that pulls it all together. You see the CRD list. Select one, and you see its pods, events, logs, and GPU utilization side by side. Restart or scale from the same interface. It is the difference between `kubectl` and `kubectl` with context.

## Key bindings

| key | action |
|---|---|
| `j`/`k` | navigate list |
| `enter` | open resource detail |
| `l` | view logs |
| `e` | view events |
| `r` | restart |
| `s` | scale |
| `g` | GPU view |
| `q` | quit |

## Install

```bash
go install github.com/santura-dev/inference-operator-tui@latest
```

## Related

- [local-inference-operator](https://github.com/santura-dev/local-inference-operator) - the operator this manages
- [vllm-logprob-tui](https://github.com/santura-dev/vllm-logprob-tui) - TUI for vLLM logprobs and token statistics
- [kubectl-tui](https://github.com/santura-dev/kubectl-tui) - general Kubernetes TUI with inference workload focus

## License

MIT
