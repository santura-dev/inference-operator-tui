// Package kube runs kubectl commands asynchronously for the TUI.
// Every method returns a tea.Cmd so the UI never blocks on the cluster.
package kube

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

const cmdTimeout = 15 * time.Second

// DeploymentInfo is one LocalInferenceService row.
type DeploymentInfo struct {
	Name   string
	Status string
}

func run(args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cmdTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, "kubectl", args...).CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("kubectl %s: %s", strings.Join(args, " "), msg)
	}
	return strings.TrimSpace(string(out)), nil
}

// ListMsg carries the LocalInferenceService list.
type ListMsg struct {
	Deployments []DeploymentInfo
	Err         error
}

// List fetches all LocalInferenceServices (name + status phase).
func List() tea.Cmd {
	return func() tea.Msg {
		out, err := run("get", "localinferenceservices", "--no-headers",
			"-o", "custom-columns=NAME:.metadata.name,STATUS:.status.phase")
		if err != nil {
			return ListMsg{Err: err}
		}
		var deps []DeploymentInfo
		for _, line := range strings.Split(out, "\n") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				deps = append(deps, DeploymentInfo{Name: parts[0], Status: parts[1]})
			}
		}
		return ListMsg{Deployments: deps}
	}
}

// ResultMsg carries output or error of an action command.
type ResultMsg struct {
	Output string
	Err    error
}

// Logs fetches the last 50 log lines for a deployment's pods.
func Logs(deployment string) tea.Cmd {
	return func() tea.Msg {
		out, err := run("logs", "-l", "app="+deployment, "--tail=50")
		if err != nil {
			return ResultMsg{Err: err}
		}
		return ResultMsg{Output: out}
	}
}

// Describe fetches the pod description for a deployment.
func Describe(deployment string) tea.Cmd {
	label := deployment
	if !strings.Contains(label, "-deployment") {
		label = deployment + "-deployment"
	}
	return func() tea.Msg {
		out, err := run("describe", "pods", "-l", "app="+label)
		if err != nil {
			return ResultMsg{Err: err}
		}
		return ResultMsg{Output: out}
	}
}

// Restart rolls the deployment's pods.
func Restart(deployment string) tea.Cmd {
	return func() tea.Msg {
		out, err := run("rollout", "restart", "deployment", deployment+"-deployment")
		if err != nil {
			return ResultMsg{Err: err}
		}
		return ResultMsg{Output: out}
	}
}

// Scale changes the replica count of a LocalInferenceService.
func Scale(deployment string, replicas int) tea.Cmd {
	return func() tea.Msg {
		out, err := run("scale", "localinferenceservice", deployment, fmt.Sprintf("--replicas=%d", replicas))
		if err != nil {
			return ResultMsg{Err: err}
		}
		return ResultMsg{Output: out}
	}
}

// Delete removes the CRD, Deployment, and Service for a name.
func Delete(deployment string) tea.Cmd {
	return func() tea.Msg {
		var errs []string
		for _, args := range [][]string{
			{"delete", "localinferenceservice", deployment},
			{"delete", "deployment", deployment + "-deployment"},
			{"delete", "service", deployment + "-service"},
		} {
			if _, err := run(args...); err != nil {
				errs = append(errs, err.Error())
			}
		}
		if len(errs) > 0 {
			return ResultMsg{Err: fmt.Errorf("%s", strings.Join(errs, "; "))}
		}
		return ResultMsg{Output: "Deleted " + deployment + " (CRD, Deployment, Service)"}
	}
}

// Apply pipes YAML to kubectl apply via stdin.
func Apply(yaml string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), cmdTimeout)
		defer cancel()
		cmd := exec.CommandContext(ctx, "kubectl", "apply", "-f", "-")
		cmd.Stdin = strings.NewReader(yaml)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return ResultMsg{Err: fmt.Errorf("kubectl apply: %s", strings.TrimSpace(string(out)))}
		}
		return ResultMsg{Output: strings.TrimSpace(string(out))}
	}
}

// PodStatus fetches the phase of a deployment's first pod.
func PodStatus(deployment string) tea.Cmd {
	return func() tea.Msg {
		out, err := run("get", "pods", "-l", "app="+deployment, "-o", "jsonpath={.items[0].status.phase}")
		if err != nil {
			return ResultMsg{Err: err}
		}
		return ResultMsg{Output: out}
	}
}
