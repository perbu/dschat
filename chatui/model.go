package chatui

import (
	"fmt"
	"github.com/anthdm/hollywood/actor"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/perbu/dschat/msg"
	"strings"
)

type Model struct {
	viewport    viewport.Model
	messages    []string
	textarea    textarea.Model
	senderStyle lipgloss.Style
	err         error
	nodePid     *actor.PID
	engine      *actor.Engine
	userPid     *actor.PID
}

type (
	errMsg  error
	chatMsg struct {
		id   string
		text string
	}
)

func InitialModel(userPid, nodePid *actor.PID, engine *actor.Engine) Model {
	ta := textarea.New()
	ta.Placeholder = "Send a message..."
	ta.Focus()

	ta.Prompt = "┃ "
	ta.CharLimit = 280

	ta.SetWidth(30)
	ta.SetHeight(3)

	// Remove cursor line styling
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.ShowLineNumbers = false
	vp := viewport.New(30, 5)
	vp.SetContent(`Welcome to the chat room!
Type a message and press Enter to send.`)

	ta.KeyMap.InsertNewline.SetEnabled(false)

	return Model{
		nodePid:     nodePid,
		userPid:     userPid,
		textarea:    ta,
		messages:    []string{},
		viewport:    vp,
		senderStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("5")),
		err:         nil,
		engine:      engine,
	}
}

func (m Model) Init() tea.Cmd {
	return textarea.Blink
}

func (m Model) Update(event tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
	)

	m.textarea, tiCmd = m.textarea.Update(event)
	m.viewport, vpCmd = m.viewport.Update(event)

	switch ev := event.(type) {
	case tea.KeyMsg:
		switch ev.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyEnter:
			// construct the message:
			// Todo: abstract this into a function which looks at the message, figures
			// out if it is a broadcast or direct message.
			chatMsg := msg.Message{
				From: m.nodePid,
				To:   "",
				Msg:  m.textarea.Value(),
			}
			// send the message to out local node for distribution
			m.engine.Send(m.nodePid, chatMsg)
			m.textarea.Reset()
		}

	case msg.Message:
		m.messages = append(m.messages, m.senderStyle.Render(ev.From.GetID())+": "+ev.Msg)
		m.viewport.SetContent(strings.Join(m.messages, "\n"))
		m.viewport.GotoBottom()

	// We handle errors just like any other message
	case errMsg:
		m.err = ev
		return m, nil
	}

	return m, tea.Batch(tiCmd, vpCmd)
}

func (m Model) View() string {
	return fmt.Sprintf(
		"%s\n\n%s",
		m.viewport.View(),
		m.textarea.View(),
	) + "\n\n"
}
