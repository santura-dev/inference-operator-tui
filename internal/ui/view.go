package ui

import "github.com/charmbracelet/lipgloss"

var (
	boxStyle   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("238")).Padding(0, 1)
	titleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)
	dimStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	menuStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("141")).Bold(true)
)

func (m Model) View() string {
	if !m.ready {
		return "Initializing..."
	}
	var content string
	switch m.page {
	case screenMenu:
		content = m.mainList.View()
	case screenNewDeployment:
		content = m.configList.View()
	case screenFieldInput:
		content = m.fieldInputView()
	case screenScale:
		content = m.optionsList.View()
	case screenYAMLPreview:
		content = boxStyle.Render(titleStyle.Render("YAML PREVIEW") + "\n\n" + m.yaml)
	case screenMonitor:
		content = m.monitorView()
	case screenEdit:
		content = m.editView()
	}
	status := ""
	if m.status != "" {
		status = "\n" + dimStyle.Render(m.status)
	}
	return content + "\n" + m.helpLine() + status
}

func (m Model) fieldInputView() string {
	switch m.focus {
	case "model", "runtime", "precision":
		return m.optionsList.View()
	case "tokens":
		return boxStyle.Render(titleStyle.Render("MAX TOKENS") + "\n\n" + m.tokenInput.View() + "\n\n" + dimStyle.Render("enter confirm · esc cancel"))
	case "replicas":
		return boxStyle.Render(titleStyle.Render("REPLICAS") + "\n\n" + m.replicaInput.View() + "\n\n" + dimStyle.Render("enter confirm · esc cancel"))
	case "lora":
		return boxStyle.Render(titleStyle.Render("LORA PATH") + "\n\n" + m.loraInput.View() + "\n\n" + dimStyle.Render("enter confirm · esc cancel"))
	case "batch":
		return boxStyle.Render(titleStyle.Render("BATCH SIZE") + "\n\n" + m.batchInput.View() + "\n\n" + dimStyle.Render("enter confirm · esc cancel"))
	}
	return ""
}

func (m Model) monitorView() string {
	if m.selected == "" {
		return m.deploymentsList.View()
	}
	body := titleStyle.Render("DEPLOYMENT "+m.selected) + "\n\n" + m.detail
	if m.logs != "" {
		body += "\n\n" + titleStyle.Render("LOGS") + "\n" + m.logs
	}
	return boxStyle.Render(body)
}

func (m Model) editView() string {
	if m.selected == "" {
		return m.deploymentsList.View()
	}
	body := titleStyle.Render("EDIT "+m.selected) + `

` + dimStyle.Render("actions") + `
  d  delete
  s  scale replicas
  r  restart
  l  view logs
  v  describe pods
  b  back`
	return boxStyle.Render(body)
}

func (m Model) helpLine() string {
	switch m.page {
	case screenMenu:
		return dimStyle.Render("↑↓ navigate · enter select · q quit")
	case screenNewDeployment:
		return dimStyle.Render("↑↓ select field · enter edit · b back · q quit")
	case screenFieldInput:
		return dimStyle.Render("enter confirm · esc back · q quit")
	case screenYAMLPreview:
		return dimStyle.Render("enter deploy · b back · q cancel")
	case screenMonitor:
		if m.selected == "" {
			return dimStyle.Render("↑↓ navigate · enter details · b back · q quit")
		}
		return dimStyle.Render("l logs · v describe · b back · q quit")
	case screenEdit:
		if m.selected == "" {
			return dimStyle.Render("↑↓ navigate · enter select · b back · q quit")
		}
		return dimStyle.Render("d delete · s scale · r restart · l logs · v describe · b back")
	case screenScale:
		return dimStyle.Render("↑↓ select replicas · enter confirm · b back")
	}
	return ""
}
