# Operator TUI

Terminal user interface for managing Local Inference Service deployments in Kubernetes.

## Features

- **Service Management**: List, create, and delete LocalInferenceService resources
- **Pod Monitoring**: View pod status, logs, and events
- **Interactive Scaling**: Scale deployments up/down
- **Cleanup Tools**: Remove orphaned resources
- **Real-time Updates**: Live status monitoring

## Screenshots

[Add screenshots here]

## Prerequisites

- Go 1.19+
- kubectl configured for cluster access
- Local Inference Operator installed

## Installation

```bash
git clone https://github.com/yourusername/operator-tui.git
cd operator-tui
go mod tidy
go build -o operator-tui
```

## Usage

```bash
./operator-tui
```

Navigate through menus to:
- View active inference services
- Create new model deployments
- Monitor pod health and logs
- Scale services
- Clean up resources

## Controls

- **↑↓**: Navigate items
- **Enter**: Select action
- **n**: New deployment
- **m**: Monitor mode
- **e**: Edit mode
- **d**: Delete
- **c**: Cleanup
- **q/Ctrl+C**: Quit

## Architecture

Built with:
- [Bubbletea](https://github.com/charmbracelet/bubbletea) - Terminal UI framework
- [Bubbles](https://github.com/charmbracelet/bubbles) - UI components
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Styling

## Workflow

1. **New Deployment**: Create LocalInferenceService CRD via guided form
2. **Monitor**: View real-time pod status and resource usage
3. **Edit**: Scale replicas, update configurations
4. **Cleanup**: Remove failed deployments and orphaned resources

## Integration

Works with the [Local Inference Operator](https://github.com/yourusername/local-inference-operator) to provide a complete management experience.

## Development

```bash
# Run in development
go run main.go

# Build optimized binary
go build -o operator-tui -ldflags="-s -w"
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Test thoroughly
5. Submit a pull request

## License

MIT License - see LICENSE file for details

## Related Projects

- [local-inference-operator](https://github.com/yourusername/local-inference-operator) - The operator
- [vllm-tui](https://github.com/yourusername/vllm-tui) - Chat interface
- [kubectl-tui](https://github.com/yourusername/kubectl-tui) - kubectl interface