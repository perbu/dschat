package sshnode

import (
	"context"
	"fmt"
	"github.com/anthdm/hollywood/actor"
	"github.com/perbu/dschat/msg"
	"log"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	bm "github.com/charmbracelet/wish/bubbletea"
	lm "github.com/charmbracelet/wish/logging"
	"github.com/muesli/termenv"
)

const (
	host = "localhost"
	port = 2222
)

type nodeState int

const (
	// NodeStateInit is the initial nodeState of the node
	NodeStateInit nodeState = iota
	// NodeStateRunning is the nodeState of the node when it is running
	NodeStateRunning
	// NodeStateStopped is the nodeState of the node when it is stopped
	NodeStateStopped
)

// SshNode contains a wish server and the list of running programs.
type SshNode struct {
	*ssh.Server
	mu        sync.Mutex
	pid       *actor.PID
	nodeState nodeState

	//progs []*tea.Program
}

func (a *SshNode) Receive(c *actor.Context) {
	switch c.Message().(type) {
	case actor.Initialized:
		a.mu.Lock()
		a.pid = c.PID()
		a.mu.Unlock()
	case actor.Started:
		a.Start()
	case actor.Stopped:
		a.Stop()
	case msg.Message:
		// Handle incoming message to the node
		// It should broadcast the message to all running programs

	}
}

func (a *SshNode) setState(state nodeState) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.nodeState = state
}
func (a *SshNode) state() nodeState {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.nodeState
}

// send dispatches a message to all running programs.
// XXX: replace with actor
func (a *SshNode) send(msg tea.Msg) {

}

func NewSshNode() actor.Receiver {
	a := new(SshNode)

	s, err := wish.NewServer(
		wish.WithAddress(fmt.Sprintf("%s:%d", host, port)),
		wish.WithHostKeyPath(".ssh/term_info_ed25519"),
		wish.WithMiddleware(
			bm.MiddlewareWithProgramHandler(a.ProgramHandler, termenv.ANSI256),
			lm.Middleware(),
		),
	)
	if err != nil {
		log.Fatalln(err)
	}
	a.Server = s
	return a
}

func (a *SshNode) Start() {
	var err error
	slog.Info("Starting SSH server", "host", host, "port", port)
	go func() {
		if err = a.ListenAndServe(); err != nil {
			log.Fatalln(err)
		}
	}()
}
func (a *SshNode) Stop() {
	slog.Info("Stopping SSH server")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() { cancel() }()
	if err := a.Shutdown(ctx); err != nil {
		log.Fatalln(err)
	}
}

// ProgramHandler returns a bubbletea program for a given session.
// it returns a new program which should be stored in the actor
func (a *SshNode) ProgramHandler(s ssh.Session) *tea.Program {
	if _, _, active := s.Pty(); !active {
		wish.Fatalln(s, "terminal is not active")
	}
	model := initialModel()
	model.SshNode = a
	model.id = s.User()

	p := tea.NewProgram(model, tea.WithOutput(s), tea.WithInput(s))

	// XXX: replace with actor
	// a.progs = append(a.progs, p)

	return p
}

type (
	errMsg  error
	chatMsg struct {
		id   string
		text string
	}
)

type model struct {
	*SshNode
	viewport    viewport.Model
	messages    []string
	id          string
	textarea    textarea.Model
	senderStyle lipgloss.Style
	err         error
}

func initialModel() model {
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

	return model{
		textarea:    ta,
		messages:    []string{},
		viewport:    vp,
		senderStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("5")),
		err:         nil,
	}
}

func (m model) Init() tea.Cmd {
	return textarea.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
	)

	m.textarea, tiCmd = m.textarea.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyEnter:
			// XXX: replace with actor
			m.SshNode.send(chatMsg{
				id:   m.id,
				text: m.textarea.Value(),
			})
			m.textarea.Reset()
		}

	case chatMsg:
		m.messages = append(m.messages, m.senderStyle.Render(msg.id)+": "+msg.text)
		m.viewport.SetContent(strings.Join(m.messages, "\n"))
		m.viewport.GotoBottom()

	// We handle errors just like any other message
	case errMsg:
		m.err = msg
		return m, nil
	}

	return m, tea.Batch(tiCmd, vpCmd)
}

func (m model) View() string {
	return fmt.Sprintf(
		"%s\n\n%s",
		m.viewport.View(),
		m.textarea.View(),
	) + "\n\n"
}
