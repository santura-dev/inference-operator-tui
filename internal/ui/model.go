package ui

import (
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/santura-dev/inference-operator-tui/internal/config"
	"github.com/santura-dev/inference-operator-tui/internal/kube"
)

// screen is the active page.
type screen int

const (
	screenMenu screen = iota
	screenNewDeployment
	screenFieldInput
	screenYAMLPreview
	screenMonitor
	screenEdit
	screenScale
)

// Model is the bubbletea application model.
type Model struct {
	mainList        list.Model
	deploymentsList list.Model
	configList      list.Model
	optionsList     list.Model
	textInput       textinput.Model
	tokenInput      textinput.Model
	replicaInput    textinput.Model
	loraInput       textinput.Model
	batchInput      textinput.Model
	spinner         spinner.Model

	page     screen
	focus    string
	cfg      config.DeploymentConfig
	status   string
	logs     string
	yaml     string
	detail   string
	selected string
	deps     []kube.DeploymentInfo
	ready    bool
}

// Item types for the list bubbles.
type menuItem struct{ title, desc string }

func (i menuItem) Title() string       { return i.title }
func (i menuItem) Description() string { return i.desc }
func (i menuItem) FilterValue() string { return i.title }

type deploymentItem struct{ name, status string }

func (i deploymentItem) Title() string       { return i.name }
func (i deploymentItem) Description() string { return i.status }
func (i deploymentItem) FilterValue() string { return i.name }

type configItem struct{ title, value string }

func (i configItem) Title() string       { return i.title }
func (i configItem) Description() string { return i.value }
func (i configItem) FilterValue() string { return i.title }

// New builds the initial model.
func New(cfg config.DeploymentConfig) Model {
	l := newList("Operator TUI", []list.Item{
		menuItem{"New Deployment", "Create a new model deployment"},
		menuItem{"Monitor Deployments", "View status and logs"},
		menuItem{"Edit Deployment", "Modify existing deployments"},
	})
	dl := newList("Deployments", nil)
	cl := newList("Configuration", []list.Item{
		configItem{"Model", cfg.Model},
		configItem{"Runtime", cfg.Runtime},
		configItem{"Precision", cfg.Precision},
		configItem{"Max Tokens", "100"},
		configItem{"Replicas", "1"},
		configItem{"LoRA Path", "(none)"},
		configItem{"Batch Size", "1"},
		configItem{"[Deploy]", "Create deployment"},
	})
	ol := newList("Options", nil)

	sp := spinner.New()
	sp.Spinner = spinner.Dot

	return Model{
		mainList:        l,
		deploymentsList: dl,
		configList:      cl,
		optionsList:     ol,
		textInput:       newInput("Enter custom model URI", 100),
		tokenInput:      newInput("Max tokens", 10),
		replicaInput:    newInput("Replicas", 5),
		loraInput:       newInput("LoRA path (optional)", 100),
		batchInput:      newInput("Batch size", 5),
		spinner:         sp,
		page:            screenMenu,
		cfg:             cfg,
	}
}

func newList(title string, items []list.Item) list.Model {
	l := list.New(items, list.NewDefaultDelegate(), 80, 20)
	l.Title = title
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	return l
}

func newInput(placeholder string, limit int) textinput.Model {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.CharLimit = limit
	return ti
}

// Init starts the blink and spinner loops.
func (m Model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.spinner.Tick)
}

// tickMsg drives the periodic deployment refresh.
type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(5*time.Second, func(time.Time) tea.Msg { return tickMsg{} })
}
