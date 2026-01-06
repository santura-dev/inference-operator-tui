package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type model struct {
	mainList           list.Model
	deploymentsList    list.Model
	configList         list.Model
	optionsList        list.Model
	textInput          textinput.Model
	tokenInput         textinput.Model
	replicaInput       textinput.Model
	loraInput          textinput.Model
	batchInput         textinput.Model
	state              string // "menu", "new_deployment", "new_deployment_input", "monitor", "yaml_preview", "edit"
	focus              string // "model", "runtime", "precision", "tokens", "replicas", "lora", "batch", "deploy"
	config             DeploymentConfig
	status             string
	spinner            spinner.Model
	deploymentStatus   string
	logs               string
	yamlPreview        string
	selectedDeployment string
	deployments        []DeploymentInfo
}

type DeploymentConfig struct {
	Model     string
	Runtime   string
	Precision string
	MaxTokens int
	Replicas  int
	LoRA      string
	BatchSize int
}

type DeploymentInfo struct {
	Name   string
	Status string
}

type menuItem struct {
	title, desc string
}

func (i menuItem) Title() string       { return i.title }
func (i menuItem) Description() string { return i.desc }
func (i menuItem) FilterValue() string { return i.title }

type deploymentItem struct {
	name, status string
}

func (i deploymentItem) Title() string       { return i.name }
func (i deploymentItem) Description() string { return i.status }
func (i deploymentItem) FilterValue() string { return i.name }

type configItem struct {
	title, value string
}

func (i configItem) Title() string       { return i.title }
func (i configItem) Description() string { return i.value }
func (i configItem) FilterValue() string { return i.title }

var (
	modelFlag   = flag.String("model", "", "Pre-select model (e.g., phi-3.5-mini)")
	runtimeFlag = flag.String("runtime", "", "Pre-select runtime (sglang or vllm)")
)

var (
	borderStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1)
	titleStyle  = lipgloss.NewStyle().Bold(true).Underline(true)
	menuStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("170")).Bold(true)
)

func initialModel() model {
	items := []list.Item{
		menuItem{title: "New Deployment", desc: "Create a new model deployment"},
		menuItem{title: "Monitor Current Deployments", desc: "View status and logs of deployments"},
		menuItem{title: "Edit Deployment", desc: "Modify existing deployments"},
	}

	l := list.New(items, list.NewDefaultDelegate(), 80, 20)
	l.Title = "Operator TUI"
	l.Styles.Title = menuStyle
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)

	dl := list.New([]list.Item{}, list.NewDefaultDelegate(), 80, 20)
	dl.Title = "Deployments"
	dl.Styles.Title = menuStyle
	dl.SetShowHelp(false)
	dl.SetShowStatusBar(false)
	dl.SetFilteringEnabled(false)

	// Create config list with initial items
	configItems := []list.Item{
		configItem{title: "Model", value: "facebook/opt-125m"},
		configItem{title: "Runtime", value: "sglang"},
		configItem{title: "Precision", value: "fp16"},
		configItem{title: "Max Tokens", value: "100"},
		configItem{title: "Replicas", value: "1"},
		configItem{title: "LoRA Path", value: "(none)"},
		configItem{title: "Batch Size", value: "1"},
		configItem{title: "[Deploy]", value: "Create deployment"},
	}
	cl := list.New(configItems, list.NewDefaultDelegate(), 80, 20)
	cl.Title = "Configuration"
	cl.Styles.Title = menuStyle
	cl.SetShowHelp(false)
	cl.SetShowStatusBar(false)
	cl.SetFilteringEnabled(false)

	ol := list.New([]list.Item{}, list.NewDefaultDelegate(), 80, 20)
	ol.Title = "Options"
	ol.Styles.Title = menuStyle
	ol.SetShowHelp(false)
	ol.SetShowStatusBar(false)
	ol.SetFilteringEnabled(false)

	ti := textinput.New()
	ti.Placeholder = "Enter custom model URI"
	ti.CharLimit = 100
	ti.Width = 50

	ti2 := textinput.New()
	ti2.Placeholder = "Max tokens (e.g., 100)"
	ti2.CharLimit = 10

	ti3 := textinput.New()
	ti3.Placeholder = "Replicas (e.g., 1)"
	ti3.CharLimit = 5

	ti4 := textinput.New()
	ti4.Placeholder = "LoRA path (optional)"
	ti4.CharLimit = 100

	ti5 := textinput.New()
	ti5.Placeholder = "Batch size (e.g., 1)"
	ti5.CharLimit = 5

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return model{
		mainList:        l,
		deploymentsList: dl,
		configList:      cl,
		optionsList:     ol,
		textInput:       ti,
		tokenInput:      ti2,
		replicaInput:    ti3,
		loraInput:       ti4,
		batchInput:      ti5,
		state:           "menu",
		focus:           "menu",
		config: DeploymentConfig{
			Model:     "facebook/opt-125m",
			Runtime:   "sglang",
			Precision: "fp16",
			MaxTokens: 100,
			Replicas:  1,
			BatchSize: 1,
		},
		spinner:            s,
		selectedDeployment: "",
		deployments:        []DeploymentInfo{},
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.spinner.Tick)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q":
			if m.state == "yaml_preview" {
				m.state = "new_deployment"
				return m, nil
			}
			return m, tea.Quit
		case "enter":
			return m.handleEnter()
		case "b":
			if m.state == "new_deployment_input" {
				m.state = "new_deployment"
				m.updateConfigList()
				return m, nil
			}
			if m.state == "monitor" && m.selectedDeployment != "" {
				m.selectedDeployment = ""
				m.logs = ""
				return m, nil
			}
			if m.state == "edit" && m.selectedDeployment != "" {
				m.selectedDeployment = ""
				return m, nil
			}
			if m.state != "menu" {
				m.state = "menu"
			}
			return m, nil
		case "esc":
			if m.state == "new_deployment_input" {
				m.state = "new_deployment"
				m.updateConfigList()
				return m, nil
			}
		case "d":
			if m.state == "edit" && m.selectedDeployment != "" {
				m.status = fmt.Sprintf("Step 1/3: Deleting %s CRD...", m.selectedDeployment)

				// Step 1: Delete LocalInferenceService CRD
				cmd1 := exec.Command("kubectl", "delete", "localinferenceservice", m.selectedDeployment)
				_, err1 := cmd1.CombinedOutput()

				if err1 != nil {
					m.status = fmt.Sprintf("Error: Failed to delete %s CRD: %v", m.selectedDeployment, err1)
					return m, nil
				}

				m.status = fmt.Sprintf("Step 2/3: Deleting %s Deployment...", m.selectedDeployment)

				// Step 2: Delete Deployment (which cascades to Pods)
				deploymentName := m.selectedDeployment + "-deployment"
				cmd2 := exec.Command("kubectl", "delete", "deployment", deploymentName)
				_, err2 := cmd2.CombinedOutput()

				if err2 != nil {
					m.status = fmt.Sprintf("Error: Failed to delete %s Deployment: %v", m.selectedDeployment, err2)
					return m, nil
				}

				m.status = fmt.Sprintf("Step 3/3: Deleting %s Service...", m.selectedDeployment)

				// Step 3: Delete Service
				serviceName := m.selectedDeployment + "-service"
				cmd3 := exec.Command("kubectl", "delete", "service", serviceName)
				_, err3 := cmd3.CombinedOutput()

				// Report results
				if err3 != nil {
					m.status = fmt.Sprintf("Error: Failed to delete %s Service: %v", m.selectedDeployment, err3)
					return m, nil
				}

				m.status = fmt.Sprintf("Success: Deleted %s (CRD, Deployment, Service)", m.selectedDeployment)
				m.selectedDeployment = ""
				return m, m.fetchDeployments()
			}
		case "c":
			if m.state == "edit" {
				m.status = "Cleaning orphaned resources..."
				deploymentName := "test-inference-sglang-deployment"
				serviceName := "test-inference-sglang-service"

				cmd1 := exec.Command("kubectl", "delete", "deployment", deploymentName)
				_, err1 := cmd1.CombinedOutput()

				cmd2 := exec.Command("kubectl", "delete", "service", serviceName)
				_, err2 := cmd2.CombinedOutput()

				if err1 != nil || err2 != nil {
					m.status = "Error: Could not delete orphaned resources"
				} else {
					m.status = "Success: Removed orphaned resources"
				}
				return m, nil
			}
		case "s":
			if m.state == "edit" && m.selectedDeployment != "" {
				m.state = "scale_options"
				m.loadScaleOptions()
				return m, nil
			}
		case "r":
			if m.state == "edit" && m.selectedDeployment != "" {
				m.status = fmt.Sprintf("Restarting %s...", m.selectedDeployment)
				cmd := exec.Command("kubectl", "rollout", "restart", "deployment", m.selectedDeployment)
				output, err := cmd.CombinedOutput()
				if err != nil {
					m.status = fmt.Sprintf("Error restarting: %v, output: %s", err, string(output))
				} else {
					m.status = fmt.Sprintf("Successfully restarted %s", m.selectedDeployment)
				}
			}
		case "l":
			if m.selectedDeployment != "" {
				name := m.selectedDeployment
				cmd := exec.Command("kubectl", "logs", "-l", fmt.Sprintf("app=%s", name), "--tail=20")
				output, err := cmd.Output()
				if err == nil {
					m.logs = string(output)
				} else {
					m.logs = fmt.Sprintf("Error fetching logs: %v", err)
				}
			} else {

				cmd := exec.Command("kubectl", "logs", "-l", "app=deployed-model", "--tail=20")
				output, err := cmd.Output()
				if err == nil {
					m.logs = string(output)
				} else {
					m.logs = fmt.Sprintf("Error fetching logs: %v", err)
				}
			}
		case "v":
			if (m.state == "edit" || m.state == "monitor") && m.selectedDeployment != "" {
				m.status = "Fetching pod description..."
				cmd := exec.Command("kubectl", "describe", "pod", m.selectedDeployment+"-deployment")
				output, err := cmd.CombinedOutput()
				if err != nil {
					m.status = fmt.Sprintf("Error: %v", err)
				} else {
					m.status = "Fetched pod details"
					m.logs = string(output)
				}
			}
		}
	case tea.WindowSizeMsg:
		h, v := lipgloss.NewStyle().GetFrameSize()
		m.mainList.SetSize(msg.Width-h, msg.Height-v-4)
		m.deploymentsList.SetSize(msg.Width-h, msg.Height-v-4)
		m.configList.SetSize(msg.Width-h, msg.Height-v-4)
		m.optionsList.SetSize(msg.Width-h, msg.Height-v-4)
	case tickMsg:
		return m, tickCmd()
	case deploymentsFetchedMsg:
		m.deploymentsList.SetItems(msg.items)
		m.deployments = msg.deployments
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	var cmd tea.Cmd
	switch m.state {
	case "menu":
		m.mainList, cmd = m.mainList.Update(msg)
	case "monitor", "edit":
		if m.selectedDeployment == "" {
			m.deploymentsList, cmd = m.deploymentsList.Update(msg)
		}
	case "new_deployment":
		m.configList, cmd = m.configList.Update(msg)
	case "scale_options":
		m.optionsList, cmd = m.optionsList.Update(msg)
	case "new_deployment_input":
		switch m.focus {
		case "model", "runtime", "precision":
			m.optionsList, cmd = m.optionsList.Update(msg)
		case "tokens":
			m.tokenInput, cmd = m.tokenInput.Update(msg)
			if val := m.tokenInput.Value(); val != "" {
				if n, err := strconv.Atoi(val); err == nil {
					m.config.MaxTokens = n
				}
			}
		case "replicas":
			m.replicaInput, cmd = m.replicaInput.Update(msg)
			if val := m.replicaInput.Value(); val != "" {
				if n, err := strconv.Atoi(val); err == nil {
					m.config.Replicas = n
				}
			}
		case "lora":
			m.loraInput, cmd = m.loraInput.Update(msg)
			m.config.LoRA = m.loraInput.Value()
		case "batch":
			m.batchInput, cmd = m.batchInput.Update(msg)
			if val := m.batchInput.Value(); val != "" {
				if n, err := strconv.Atoi(val); err == nil {
					m.config.BatchSize = n
				}
			}
		}
	}
	return m, cmd
}

func (m *model) updateConfigList() {
	tokenVal := strconv.Itoa(m.config.MaxTokens)
	if m.tokenInput.Value() != "" {
		tokenVal = m.tokenInput.Value()
	}

	replicaVal := strconv.Itoa(m.config.Replicas)
	if m.replicaInput.Value() != "" {
		replicaVal = m.replicaInput.Value()
	}

	batchVal := strconv.Itoa(m.config.BatchSize)
	if m.batchInput.Value() != "" {
		batchVal = m.batchInput.Value()
	}

	loraVal := m.config.LoRA
	if loraVal == "" {
		loraVal = "(none)"
	}

	items := []list.Item{
		configItem{title: "Model", value: m.config.Model},
		configItem{title: "Runtime", value: m.config.Runtime},
		configItem{title: "Precision", value: m.config.Precision},
		configItem{title: "Max Tokens", value: tokenVal},
		configItem{title: "Replicas", value: replicaVal},
		configItem{title: "LoRA Path", value: loraVal},
		configItem{title: "Batch Size", value: batchVal},
		configItem{title: "[Deploy]", value: "Create deployment"},
	}
	m.configList.SetItems(items)
}

func (m model) handleEnter() (tea.Model, tea.Cmd) {
	switch m.state {
	case "menu":
		if len(m.mainList.Items()) == 0 {
			return m, nil
		}
		selected := m.mainList.SelectedItem()
		if selected == nil {
			return m, nil
		}
		menuSelected := selected.(menuItem)
		switch menuSelected.title {
		case "New Deployment":
			m.state = "new_deployment"
			m.focus = "model"
			m.updateConfigList()
		case "Monitor Current Deployments":
			m.state = "monitor"
			m.selectedDeployment = ""
			return m, m.fetchDeployments()
		case "Edit Deployment":
			m.state = "edit"
			m.selectedDeployment = ""
			return m, m.fetchDeployments()
		}
	case "yaml_preview":
		m.status = "Saving YAML..."
		err := saveYAML(m.yamlPreview)
		if err != nil {
			m.status = fmt.Sprintf("Error saving YAML: %v", err)
		} else {
			m.status = "Applying to Kubernetes..."
			err := applyYAML()
			if err != nil {
				m.status = fmt.Sprintf("Error applying: %v", err)
			} else {
				m.status = "Deployment successful! Monitoring status..."
				m.state = "menu"
				return m, tickCmd()
			}
		}
	case "new_deployment":
		if len(m.configList.Items()) == 0 {
			return m, nil
		}
		selected := m.configList.SelectedItem()
		if selected == nil {
			return m, nil
		}
		configSelected := selected.(configItem)
		switch configSelected.title {
		case "[Deploy]":
			yaml := generateYAML(m.config)
			m.yamlPreview = yaml
			m.state = "yaml_preview"
		case "Model":
			m.state = "new_deployment_input"
			m.focus = "model"
			m.loadModelOptions()
		case "Runtime":
			m.state = "new_deployment_input"
			m.focus = "runtime"
			m.loadRuntimeOptions()
		case "Precision":
			m.state = "new_deployment_input"
			m.focus = "precision"
			m.loadPrecisionOptions()
		case "Max Tokens":
			m.state = "new_deployment_input"
			m.focus = "tokens"
			m.tokenInput.SetValue(strconv.Itoa(m.config.MaxTokens))
			m.tokenInput.Focus()
		case "Replicas":
			m.state = "new_deployment_input"
			m.focus = "replicas"
			m.replicaInput.SetValue(strconv.Itoa(m.config.Replicas))
			m.replicaInput.Focus()
		case "LoRA Path":
			m.state = "new_deployment_input"
			m.focus = "lora"
			m.loraInput.SetValue(m.config.LoRA)
			m.loraInput.Focus()
		case "Batch Size":
			m.state = "new_deployment_input"
			m.focus = "batch"
			m.batchInput.SetValue(strconv.Itoa(m.config.BatchSize))
			m.batchInput.Focus()
		}
	case "new_deployment_input":
		switch m.focus {
		case "model":
			if len(m.optionsList.Items()) == 0 {
				return m, nil
			}
			selected := m.optionsList.SelectedItem()
			if selected == nil {
				return m, nil
			}
			menuSelected := selected.(menuItem)
			m.config.Model = menuSelected.title
			m.state = "new_deployment"
			m.updateConfigList()
		case "runtime":
			if len(m.optionsList.Items()) == 0 {
				return m, nil
			}
			selected := m.optionsList.SelectedItem()
			if selected == nil {
				return m, nil
			}
			menuSelected := selected.(menuItem)
			m.config.Runtime = menuSelected.title
			m.state = "new_deployment"
			m.updateConfigList()
		case "precision":
			if len(m.optionsList.Items()) == 0 {
				return m, nil
			}
			selected := m.optionsList.SelectedItem()
			if selected == nil {
				return m, nil
			}
			menuSelected := selected.(menuItem)
			m.config.Precision = menuSelected.title
			m.state = "new_deployment"
			m.updateConfigList()
		case "tokens":
			if val := m.tokenInput.Value(); val != "" {
				if n, err := strconv.Atoi(val); err == nil {
					m.config.MaxTokens = n
				}
			}
			m.state = "new_deployment"
			m.updateConfigList()
		case "replicas":
			if val := m.replicaInput.Value(); val != "" {
				if n, err := strconv.Atoi(val); err == nil {
					m.config.Replicas = n
				}
			}
			m.state = "new_deployment"
			m.updateConfigList()
		case "lora":
			m.config.LoRA = m.loraInput.Value()
			m.state = "new_deployment"
			m.updateConfigList()
		case "batch":
			if val := m.batchInput.Value(); val != "" {
				if n, err := strconv.Atoi(val); err == nil {
					m.config.BatchSize = n
				}
			}
			m.state = "new_deployment"
			m.updateConfigList()
		}
	case "monitor", "edit":
		if m.selectedDeployment == "" {
			if len(m.deploymentsList.Items()) == 0 {
				return m, nil
			}
			selected := m.deploymentsList.SelectedItem()
			if selected == nil {
				return m, nil
			}
			deploySelected := selected.(deploymentItem)
			m.selectedDeployment = deploySelected.name
			m.pollStatusForSelected()
		}
	case "scale_options":
		if len(m.optionsList.Items()) == 0 {
			return m, nil
		}
		selected := m.optionsList.SelectedItem()
		if selected == nil {
			return m, nil
		}
		scaleSelected := selected.(menuItem)
		m.status = fmt.Sprintf("Scaling %s to %s replicas...", m.selectedDeployment, scaleSelected.title)
		cmd := exec.Command("kubectl", "scale", "localinferenceservice", m.selectedDeployment, "--replicas="+scaleSelected.title)
		output, err := cmd.CombinedOutput()
		if err != nil {
			m.status = fmt.Sprintf("Error scaling: %v, output: %s", err, string(output))
		} else {
			m.status = fmt.Sprintf("Successfully scaled %s to %s replicas", m.selectedDeployment, scaleSelected.title)
		}
		m.state = "edit"
		return m, m.fetchDeployments()
	}
	return m, nil
}

func (m model) View() string {
	content := ""
	switch m.state {
	case "menu":
		content = m.mainList.View()
	case "new_deployment":
		content = m.configList.View()
	case "new_deployment_input":
		switch m.focus {
		case "model", "runtime", "precision":
			content = m.optionsList.View()
		case "tokens":
			content = borderStyle.Render(fmt.Sprintf("Max Tokens\n\n%s\n\nPress Enter to confirm, esc to cancel", m.tokenInput.View()))
		case "replicas":
			content = borderStyle.Render(fmt.Sprintf("Replicas\n\n%s\n\nPress Enter to confirm, esc to cancel", m.replicaInput.View()))
		case "lora":
			content = borderStyle.Render(fmt.Sprintf("LoRA Path\n\n%s\n\nPress Enter to confirm, esc to cancel", m.loraInput.View()))
		case "batch":
			content = borderStyle.Render(fmt.Sprintf("Batch Size\n\n%s\n\nPress Enter to confirm, esc to cancel", m.batchInput.View()))
		}

	case "scale_options":
		content = m.optionsList.View()
	case "yaml_preview":
		content = borderStyle.Render("YAML Preview\n\n" + m.yamlPreview)
	case "monitor":
		if m.selectedDeployment == "" {
			content = m.deploymentsList.View()
		} else {
			content = borderStyle.Render(fmt.Sprintf("Deployment: %s\n\nStatus: %s\n\nPod Status: %s\n\nRecent Logs:\n%s",
				m.selectedDeployment, m.getDeploymentStatus(m.selectedDeployment), m.deploymentStatus, m.logs))
		}
	case "edit":
		if m.selectedDeployment == "" {
			content = m.deploymentsList.View()
		} else {
			// Show edit options
			editContent := fmt.Sprintf("Edit Deployment: %s\n\nStatus: %s\n\n", m.selectedDeployment, m.getDeploymentStatus(m.selectedDeployment))
			editContent += "Available Actions:\n"
			editContent += "• [d] Delete deployment\n"
			editContent += "• [s] Scale replicas\n"
			editContent += "• [r] Restart deployment\n"
			editContent += "• [l] View logs\n"
			editContent += "• [v] View pod details/describe\n"
			editContent += "• [c] Cleanup orphaned resources\n"
			editContent += "\nPress a key in brackets to perform action"
			content = borderStyle.Render(editContent)
		}
	default:
		content = "Unknown state"
	}

	statusLine := ""
	if m.status != "" {
		statusLine = "\n" + m.status
	}

	return fmt.Sprintf("%s\n%s%s", content, m.renderHelp(), statusLine)
}

func (m model) renderHelp() string {
	switch m.state {
	case "menu":
		return "↑↓: navigate | Enter: select | q: quit"
	case "new_deployment":
		return "↑↓: select field | Enter: edit | b: back to menu | q: quit"
	case "new_deployment_input":
		return "↑↓: select option | Enter: confirm | esc: cancel | q: quit"
	case "yaml_preview":
		return "Enter: deploy | q: cancel | b: back"
	case "monitor":
		if m.selectedDeployment == "" {
			return "↑↓: navigate | Enter: view details | b: back to menu | q: quit"
		}
		return "l: refresh logs | v: view pod details | b: back | q: quit"
	case "edit":
		if m.selectedDeployment == "" {
			return "↑↓: navigate | Enter: select | b: back to menu | q: quit"
		}
		return "d: delete | s: scale | r: restart | l: logs | v: describe | c: cleanup | b: back | q: quit"
	case "scale_options":
		return "↑↓: select replicas | Enter: confirm | b: back | q: quit"
	case "view_describe":
		return "b: back | q: quit"
	default:
		return ""
	}
}

func (m *model) loadModelOptions() {
	items := []list.Item{
		menuItem{title: "facebook/opt-125m", desc: "Small test model"},
		menuItem{title: "Phi-3.5-mini", desc: "Microsoft Phi 3.5 Mini"},
		menuItem{title: "Qwen2.5-3B", desc: "Qwen 2.5 3B"},
		menuItem{title: "Gemma-2-2B", desc: "Google Gemma 2 2B"},
		menuItem{title: "Custom", desc: "Enter custom model URI"},
	}
	m.optionsList.SetItems(items)
}

func (m *model) loadRuntimeOptions() {
	items := []list.Item{
		menuItem{title: "sglang", desc: "SGLang runtime"},
		menuItem{title: "vllm", desc: "vLLM runtime"},
	}
	m.optionsList.SetItems(items)
}

func (m *model) loadPrecisionOptions() {
	items := []list.Item{
		menuItem{title: "fp16", desc: "Half precision (float16)"},
		menuItem{title: "fp32", desc: "Full precision (float32)"},
		menuItem{title: "int8", desc: "Integer 8-bit"},
	}
	m.optionsList.SetItems(items)
}

func (m *model) loadScaleOptions() {
	items := []list.Item{
		menuItem{title: "0", desc: "Stop deployment (scale to 0)"},
		menuItem{title: "1", desc: "Single replica"},
		menuItem{title: "2", desc: "Two replicas"},
		menuItem{title: "5", desc: "Five replicas"},
		menuItem{title: "10", desc: "Ten replicas"},
	}
	m.optionsList.SetItems(items)
}

func (m model) getDeploymentStatus(name string) string {
	for _, dep := range m.deployments {
		if dep.Name == name {
			return dep.Status
		}
	}
	return "Unknown"
}

func generateYAML(config DeploymentConfig) string {
	yaml := fmt.Sprintf(`apiVersion: serving.local-ome.com/v1
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
    replicas: %d`, config.Runtime, config.Model, config.BatchSize, config.Precision, config.MaxTokens, config.Replicas)
	if config.LoRA != "" {
		yaml += fmt.Sprintf(`
  lora:
    loraPath: "%s"`, config.LoRA)
	}
	return yaml
}

func saveYAML(yaml string) error {
	return os.WriteFile("deployment.yaml", []byte(yaml), 0644)
}

func applyYAML() error {
	cmd := exec.Command("kubectl", "apply", "-f", "deployment.yaml")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("kubectl apply failed: %v, output: %s", err, string(output))
	}
	return nil
}

type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second*5, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m *model) pollStatus() {
	cmd := exec.Command("kubectl", "get", "pods", "-l", "app=deployed-model", "-o", "jsonpath={.items[0].status.phase}")
	output, err := cmd.Output()
	if err == nil {
		m.deploymentStatus = strings.TrimSpace(string(output))
	} else {
		m.deploymentStatus = "Error fetching status"
	}
}

type deploymentsFetchedMsg struct {
	items       []list.Item
	deployments []DeploymentInfo
	error       error
}

func (m *model) fetchDeployments() tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("kubectl", "get", "localinferenceservices", "--no-headers", "-o", "custom-columns=NAME:.metadata.name,STATUS:.status.phase")
		output, err := cmd.Output()
		var items []list.Item
		deployments := []DeploymentInfo{}
		if err == nil {
			outputStr := strings.TrimSpace(string(output))
			if outputStr != "" {
				lines := strings.Split(outputStr, "\n")
				for _, line := range lines {
					parts := strings.Fields(line)
					if len(parts) >= 2 {
						deployments = append(deployments, DeploymentInfo{Name: parts[0], Status: parts[1]})
						items = append(items, deploymentItem{name: parts[0], status: parts[1]})
					}
				}
			}
		}
		return deploymentsFetchedMsg{items: items, deployments: deployments, error: err}
	}
}

func (m *model) pollStatusForSelected() {
	if m.selectedDeployment != "" {
		cmd := exec.Command("bash", "-c", fmt.Sprintf("kubectl get pods -l app=%s -o 'jsonpath={.items[0].status.phase}'", m.selectedDeployment))
		output, err := cmd.Output()
		if err == nil {
			m.deploymentStatus = strings.TrimSpace(string(output))
		} else {
			m.deploymentStatus = "Error fetching status"
		}
	}
}

func main() {
	flag.Parse()

	m := initialModel()
	if *modelFlag != "" {
		m.config.Model = *modelFlag
		m.state = "new_deployment"
		m.focus = "runtime"
	}
	if *runtimeFlag != "" {
		m.config.Runtime = *runtimeFlag
		m.state = "new_deployment"
		m.focus = "deploy"
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}
