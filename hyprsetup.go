package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type model struct {
	choices      []string
	cursor       int
	selected     string
	quitting     bool
	logs         []string
	progress     string
	isProcessing bool
}

type statusMsg struct {
	status string
	err    error
}

type configOption struct {
	name  string
	value string
}

func initialModel() model {
	return model{
		choices: []string{"Install Hyprland", "Configure Hyprland", "Troubleshoot", "Save Logs", "Exit"},
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		tea.Tick(time.Second*1, func(t time.Time) tea.Msg { return nil }),
		func() tea.Msg { return statusMsg{status: "Initializing Hyprland Setup..."} },
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}
		case "enter":
			m.selected = m.choices[m.cursor]
			m.isProcessing = true

			switch m.selected {
			case "Install Hyprland":
				return m, installPackagesOneByOne()
			case "Configure Hyprland":
				return m, configureHyprland()
			case "Troubleshoot":
				return m, troubleshootHyprland()
			case "Save Logs":
				return m, saveLogsToFile(m)
			case "Exit":
				m.quitting = true
				return m, tea.Quit
			}
		}
	case statusMsg:
		m.logs = append(m.logs, msg.status)
		m.isProcessing = false
		if msg.err != nil {
			m.logs = append(m.logs, fmt.Sprintf("Error: %v", msg.err))
		}
		return m, nil
	}

	return m, nil
}

func (m model) View() string {
	s := "Hyprland Setup Assistant for FreeBSD\n\n"

	for i, choice := range m.choices {
		cursor := " "
		if m.cursor == i {
			cursor = ">" // cursor for the selected option
		}
		s += fmt.Sprintf("%s %s\n", cursor, choice)
	}

	if m.selected != "" {
		s += "\nSelected: " + m.selected + "\n"
		if m.isProcessing {
			s += "Processing" + strings.Repeat(".", len(m.progress)%4) + "\n"
		}
		for _, log := range m.logs {
			s += lipgloss.NewStyle().Foreground(lipgloss.Color("#FFA07A")).Render(log + "\n")
		}
	}

	if m.quitting {
		s += "\nExiting..."
	}

	return lipgloss.NewStyle().Padding(1, 2).Render(s)
}

// Install packages one by one and show progress for each package
func installPackagesOneByOne() tea.Cmd {
	return func() tea.Msg {
		pkgs := []string{"hyprland", "wlroots", "xwayland", "waybar", "grim", "jq", "wofi", "alacritty", "pam_xdg", "hyprpaper"}
		for _, pkg := range pkgs {
			cmd := exec.Command("sudo", "pkg", "install", "-y", pkg)
			out, err := cmd.CombinedOutput()
			if err != nil {
				return statusMsg{status: fmt.Sprintf("Failed to install %s", pkg), err: fmt.Errorf(string(out))}
			}
			time.Sleep(500 * time.Millisecond) // Simulate install time for visual feedback
			fmt.Printf("Successfully installed %s\n", pkg) // Output to console for each successful install
		}

		// Copy the provided hyprland.conf to the correct location
		homeDir, _ := os.UserHomeDir()
		configDir := filepath.Join(homeDir, ".config", "hypr")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return statusMsg{status: "Failed to create hypr configuration directory", err: err}
		}

		// Copy the hyprland.conf from the current working directory
		confPath := "hyprland.conf" // Assumes this file is in the current directory
		destPath := filepath.Join(configDir, "hyprland.conf")
		input, err := os.ReadFile(confPath)
		if err != nil {
			return statusMsg{status: "Failed to read hyprland.conf", err: err}
		}
		if err := os.WriteFile(destPath, input, 0644); err != nil {
			return statusMsg{status: "Failed to write hyprland.conf", err: err}
		}

		return statusMsg{status: "Hyprland installation and configuration completed successfully."}
	}
}

func configureHyprland() tea.Cmd {
	return func() tea.Msg {
		// Configuration process is covered during installation in this case
		return statusMsg{status: "Configuration completed."}
	}
}

func troubleshootHyprland() tea.Cmd {
	return func() tea.Msg {
		checks := []func() statusMsg{
			checkHyprlandInstalled,
			checkXDGEnv,
		}
		for _, check := range checks {
			result := check()
			if result.err != nil {
				return result
			}
		}
		return statusMsg{status: "No common issues found."}
	}
}

func checkHyprlandInstalled() statusMsg {
	_, err := exec.LookPath("Hyprland")
	if err != nil {
		return statusMsg{status: "Hyprland not found in PATH", err: err}
	}
	return statusMsg{status: "Hyprland is installed."}
}

func checkXDGEnv() statusMsg {
	xdg := os.Getenv("XDG_RUNTIME_DIR")
	if xdg != "/tmp" {
		return statusMsg{status: fmt.Sprintf("XDG_RUNTIME_DIR is set to %s, expected /tmp", xdg)}
	}
	return statusMsg{status: "XDG_RUNTIME_DIR is correctly set to /tmp."}
}

func saveLogsToFile(m model) tea.Cmd {
	return func() tea.Msg {
		logFile := filepath.Join(os.TempDir(), "hyprland_setup.log")
		file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return statusMsg{status: "Failed to open log file for writing", err: err}
		}
		defer file.Close()

		for _, log := range m.logs {
			if _, err := file.WriteString(log + "\n"); err != nil {
				return statusMsg{status: "Failed to write to log file", err: err}
			}
		}
		return statusMsg{status: fmt.Sprintf("Logs saved to %s", logFile)}
	}
}

func setupEnvironment() {
	os.Setenv("XDG_RUNTIME_DIR", "/tmp")
	if _, err := os.Stat("/tmp"); os.IsNotExist(err) {
		if err := os.Mkdir("/tmp", 0700); err != nil {
			log.Println("Failed to create /tmp:", err)
		}
	}
}

func main() {
	setupEnvironment()
	p := tea.NewProgram(initialModel())
	if err := p.Start(); err != nil {
		log.Fatalf("Alas, there's been an error: %v", err)
	}
}

