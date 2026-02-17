package ui

import (
	"strconv"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/santura-dev/inference-operator-tui/internal/kube"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		w, h := msg.Width, msg.Height-4
		m.mainList.SetSize(w, h)
		m.deploymentsList.SetSize(w, h)
		m.configList.SetSize(w, h)
		m.optionsList.SetSize(w, h)
		m.ready = true
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case tickMsg:
		if m.page == screenMonitor || m.page == screenEdit {
			return m, tea.Batch(kube.List(), tickCmd())
		}
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case kube.ListMsg:
		items := make([]list.Item, 0, len(msg.Deployments))
		for _, d := range msg.Deployments {
			items = append(items, deploymentItem{d.Name, d.Status})
		}
		m.deploymentsList.SetItems(items)
		m.deps = msg.Deployments
		if msg.Err != nil {
			m.status = msg.Err.Error()
		}
		return m, nil

	case kube.ResultMsg:
		if msg.Err != nil {
			m.status = "error: " + msg.Err.Error()
			return m, nil
		}
		m.status = msg.Output
		if m.selected != "" && (m.page == screenMonitor) {
			m.logs = msg.Output
			m.detail = msg.Output
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.forwardToLists(msg)
	return m, cmd
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "q":
		if m.page == screenYAMLPreview {
			m.page = screenNewDeployment
			return m, nil
		}
		return m, tea.Quit
	case "b", "esc":
		return m.back()
	case "enter":
		return m.handleEnter()
	case "d":
		if m.page == screenEdit && m.selected != "" {
			m.status = "Deleting " + m.selected + "..."
			return m, kube.Delete(m.selected)
		}
	case "s":
		if m.page == screenEdit && m.selected != "" {
			m.page = screenScale
			m.optionsList.SetItems([]list.Item{
				menuItem{"0", "stop (scale to zero)"},
				menuItem{"1", "single replica"},
				menuItem{"2", "two replicas"},
				menuItem{"5", "five replicas"},
				menuItem{"10", "ten replicas"},
			})
			return m, nil
		}
	case "r":
		if m.page == screenEdit && m.selected != "" {
			m.status = "Restarting " + m.selected + "..."
			return m, kube.Restart(m.selected)
		}
	case "l":
		if m.selected != "" {
			m.status = "Fetching logs..."
			return m, kube.Logs(m.selected)
		}
	case "v":
		if m.selected != "" {
			m.status = "Fetching pod description..."
			return m, kube.Describe(m.selected)
		}
	}
	return m, nil
}

func (m Model) back() (tea.Model, tea.Cmd) {
	switch m.page {
	case screenFieldInput:
		m.page = screenNewDeployment
		m.updateConfigList()
	case screenYAMLPreview:
		m.page = screenNewDeployment
	case screenScale:
		m.page = screenEdit
	case screenMonitor, screenEdit:
		if m.selected != "" {
			m.selected = ""
			m.logs = ""
			m.detail = ""
		} else {
			m.page = screenMenu
		}
	case screenNewDeployment:
		m.page = screenMenu
	}
	return m, nil
}

func (m Model) handleEnter() (tea.Model, tea.Cmd) {
	switch m.page {
	case screenMenu:
		item, ok := m.mainList.SelectedItem().(menuItem)
		if !ok {
			return m, nil
		}
		switch item.title {
		case "New Deployment":
			m.page = screenNewDeployment
			m.updateConfigList()
		case "Monitor Deployments":
			m.page = screenMonitor
			m.selected = ""
			return m, tea.Batch(kube.List(), tickCmd())
		case "Edit Deployment":
			m.page = screenEdit
			m.selected = ""
			return m, tea.Batch(kube.List(), tickCmd())
		}

	case screenYAMLPreview:
		m.status = "Applying to Kubernetes..."
		return m, kube.Apply(m.yaml)

	case screenNewDeployment:
		item, ok := m.configList.SelectedItem().(configItem)
		if !ok {
			return m, nil
		}
		switch item.title {
		case "[Deploy]":
			m.yaml = kube.GenerateYAML(m.cfg)
			m.page = screenYAMLPreview
		case "Model":
			m.page = screenFieldInput
			m.focus = "model"
			m.optionsList.SetItems([]list.Item{
				menuItem{"facebook/opt-125m", "small test model"},
				menuItem{"Phi-3.5-mini", "Microsoft Phi 3.5 Mini"},
				menuItem{"Qwen2.5-3B", "Qwen 2.5 3B"},
				menuItem{"Gemma-2-2B", "Google Gemma 2 2B"},
				menuItem{"Custom", "enter custom model URI"},
			})
		case "Runtime":
			m.page = screenFieldInput
			m.focus = "runtime"
			m.optionsList.SetItems([]list.Item{
				menuItem{"sglang", "SGLang runtime"},
				menuItem{"vllm", "vLLM runtime"},
			})
		case "Precision":
			m.page = screenFieldInput
			m.focus = "precision"
			m.optionsList.SetItems([]list.Item{
				menuItem{"fp16", "half precision"},
				menuItem{"fp32", "full precision"},
				menuItem{"int8", "8-bit integer"},
			})
		case "Max Tokens":
			m.page, m.focus = screenFieldInput, "tokens"
			m.tokenInput.SetValue(strconv.Itoa(m.cfg.MaxTokens))
			m.tokenInput.Focus()
		case "Replicas":
			m.page, m.focus = screenFieldInput, "replicas"
			m.replicaInput.SetValue(strconv.Itoa(m.cfg.Replicas))
			m.replicaInput.Focus()
		case "LoRA Path":
			m.page, m.focus = screenFieldInput, "lora"
			m.loraInput.SetValue(m.cfg.LoRA)
			m.loraInput.Focus()
		case "Batch Size":
			m.page, m.focus = screenFieldInput, "batch"
			m.batchInput.SetValue(strconv.Itoa(m.cfg.BatchSize))
			m.batchInput.Focus()
		}

	case screenFieldInput:
		switch m.focus {
		case "model", "runtime", "precision":
			item, ok := m.optionsList.SelectedItem().(menuItem)
			if !ok {
				return m, nil
			}
			switch m.focus {
			case "model":
				m.cfg.Model = item.title
			case "runtime":
				m.cfg.Runtime = item.title
			case "precision":
				m.cfg.Precision = item.title
			}
			m.page = screenNewDeployment
			m.updateConfigList()
		default:
			m.applyFieldInput()
			m.page = screenNewDeployment
			m.updateConfigList()
		}

	case screenMonitor, screenEdit:
		if m.selected == "" {
			item, ok := m.deploymentsList.SelectedItem().(deploymentItem)
			if !ok {
				return m, nil
			}
			m.selected = item.name
			return m, kube.PodStatus(item.name)
		}

	case screenScale:
		item, ok := m.optionsList.SelectedItem().(menuItem)
		if !ok {
			return m, nil
		}
		n, err := strconv.Atoi(item.title)
		if err != nil {
			return m, nil
		}
		m.status = "Scaling " + m.selected + "..."
		m.page = screenEdit
		return m, tea.Batch(kube.Scale(m.selected, n), kube.List())
	}
	return m, nil
}

// applyFieldInput reads the focused text input into config.
func (m *Model) applyFieldInput() {
	switch m.focus {
	case "tokens":
		if n, err := strconv.Atoi(m.tokenInput.Value()); err == nil {
			m.cfg.MaxTokens = n
		}
	case "replicas":
		if n, err := strconv.Atoi(m.replicaInput.Value()); err == nil {
			m.cfg.Replicas = n
		}
	case "lora":
		m.cfg.LoRA = m.loraInput.Value()
	case "batch":
		if n, err := strconv.Atoi(m.batchInput.Value()); err == nil {
			m.cfg.BatchSize = n
		}
	}
}

func (m *Model) updateConfigList() {
	lora := m.cfg.LoRA
	if lora == "" {
		lora = "(none)"
	}
	m.configList.SetItems([]list.Item{
		configItem{"Model", m.cfg.Model},
		configItem{"Runtime", m.cfg.Runtime},
		configItem{"Precision", m.cfg.Precision},
		configItem{"Max Tokens", strconv.Itoa(m.cfg.MaxTokens)},
		configItem{"Replicas", strconv.Itoa(m.cfg.Replicas)},
		configItem{"LoRA Path", lora},
		configItem{"Batch Size", strconv.Itoa(m.cfg.BatchSize)},
		configItem{"[Deploy]", "Create deployment"},
	})
}

// forwardToLists routes non-intercepted messages to the active list.
func (m *Model) forwardToLists(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	switch m.page {
	case screenMenu:
		m.mainList, cmd = m.mainList.Update(msg)
	case screenMonitor, screenEdit:
		if m.selected == "" {
			m.deploymentsList, cmd = m.deploymentsList.Update(msg)
		}
	case screenNewDeployment:
		m.configList, cmd = m.configList.Update(msg)
	case screenFieldInput, screenScale:
		m.optionsList, cmd = m.optionsList.Update(msg)
	}
	return cmd
}
