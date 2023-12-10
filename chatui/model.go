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
	"time"
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
			m.engine.Send(m.userPid, msg.UserDisconnected{})
			return m, tea.Quit
		case tea.KeyEnter:
			// construct the message:
			// Todo: abstract this into a function which looks at the message, figures
			// out if it is a broadcast or direct message.

			chatMsg, err := m.NewEgressMessage(m.textarea.Value())
			if err != nil {
				fmt.Println("error:", err)
				m.textarea.Reset()
				break
			}
			// send the message to out local node for distribution
			m.engine.Send(m.userPid, chatMsg)
			m.textarea.Reset()
		}

	case msg.IngressMessage:
		timeStamp := time.Now().Format("15:04:05")
		m.messages = append(m.messages, m.senderStyle.Render(ev.From.ID)+timeStamp+": "+ev.Msg)
		m.viewport.SetContent(strings.Join(m.messages, "\n"))
		m.viewport.GotoBottom()

	// We handle errors just like any other message
	case errMsg:
		m.err = ev
		return m, nil
	}

	return m, tea.Batch(tiCmd, vpCmd)
}

func (m Model) NewEgressMessage(command string) (msg.EgressMessage, error) {
	recipient := "" // default to broadcast
	// if it starts with "/" it is a command:
	if strings.HasPrefix(command, "/msg") {
		tokens := strings.Split(command, " ")
		if len(tokens) < 3 {
			return msg.EgressMessage{}, fmt.Errorf("invalid command")
		}
		recipient = tokens[1]
		// this kinda alters the message, but we can fix that later (never)
		command = strings.Join(tokens[2:], " ")
	}

	return msg.EgressMessage{
		From: m.userPid,
		To:   recipient,
		Msg:  command,
	}, nil
}

func (m Model) View() string {
	return fmt.Sprintf(
		"%s\n\n%s",
		m.viewport.View(),
		m.textarea.View(),
	) + "\n\n"
}
