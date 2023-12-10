package sshnode

import (
	"context"
	"fmt"
	"github.com/anthdm/hollywood/actor"
	"github.com/perbu/dschat/chatui"
	"github.com/perbu/dschat/msg"
	"log"
	"log/slog"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
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
	sshServer *ssh.Server
	mu        sync.Mutex
	pid       *actor.PID
	nodeState nodeState
	engine    *actor.Engine
	children  []*actor.PID
}

func (a *SshNode) Receive(c *actor.Context) {
	switch c.Message().(type) {
	case actor.Initialized:
		a.mu.Lock()
		a.pid = c.PID()
		a.engine = c.Engine()
		a.mu.Unlock()
	case actor.Started:
		a.Start()
	case actor.Stopped:
		a.Stop()
	case NewUserRequest:
		// wish server has a new user
		// create a new user actor
		// check if the user is already connected
		m := c.Message().(NewUserRequest)
		eChild := c.Child(m.userid)
		if eChild != nil {
			c.Send(c.Sender(), NewUserResponse{err: fmt.Errorf("user %s already connected", m.userid)})
			return
		}
		userPid := c.SpawnChild(newUser(m.userid), m.userid)
		a.addChile(userPid)
		c.Send(c.Sender(), NewUserResponse{
			nodePid: c.PID(),
			userPid: userPid,
		})
	case msg.UserDisconnected:
		// wish server has a user that has disconnected
		// find the user actor and stop it
		a.removeChild(c.Sender())
		a.engine.Poison(c.Sender()).Wait()
	case msg.EgressMessage:
		m := c.Message().(msg.EgressMessage)
		// broadcast the message to all users:
		for _, p := range a.children {
			fmt.Printf("NODE: %s --> %s\n", m.From.GetID(), p.GetID())
			im := m.ToIngress()
			a.engine.Send(p, im)
		}
	default:
		fmt.Printf("Node got an unknown message: %T\n", c.Message())
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
func (a *SshNode) addChile(pid *actor.PID) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.children = append(a.children, pid)
}
func (a *SshNode) removeChild(pid *actor.PID) {
	a.mu.Lock()
	defer a.mu.Unlock()
	for i, p := range a.children {
		if p == pid {
			a.children = append(a.children[:i], a.children[i+1:]...)
			return
		}
	}
}

// send dispatches a message to all running programs.
// XXX: replace with actor
func (a *SshNode) send(msg tea.Msg) {
	slog.Info("node.send", "msg", msg)
}

func NewSshNode() actor.Receiver {
	a := &SshNode{
		nodeState: NodeStateInit,
		children:  make([]*actor.PID, 0),
	}

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
	a.sshServer = s
	return a
}

func (a *SshNode) Start() {
	state := a.state()
	if state == NodeStateRunning {
		return
	}
	a.setState(NodeStateRunning)
	var err error
	slog.Info("Starting SSH server", "host", host, "port", port)
	go func() {
		if err = a.sshServer.ListenAndServe(); err != nil {
			log.Fatalln(err)
		}
	}()
}
func (a *SshNode) Stop() {
	state := a.state()
	if state == NodeStateStopped {
		return
	}
	a.setState(NodeStateStopped)
	slog.Info("Stopping SSH server")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() { cancel() }()
	if err := a.sshServer.Shutdown(ctx); err != nil {
		log.Fatalln(err)
	}
}

// ProgramHandler is launched by Wish Middleware when a new connection is established.
// it returns a new program which should be stored in the actor so it can be
// used by the Receive method.
func (a *SshNode) ProgramHandler(s ssh.Session) *tea.Program {
	if _, _, active := s.Pty(); !active {
		wish.Fatalln(s, "terminal is not active")
	}
	resp, err := a.engine.Request(a.pid,
		NewUserRequest{
			userid: s.User(),
		}, time.Millisecond*100).Result()
	if err != nil {
		wish.Fatalln(s, "error:", err)
	}
	res, ok := resp.(NewUserResponse)
	if !ok {
		wish.Fatalln(s, "error: invalid response")
	}
	if res.err != nil {
		wish.Fatalln(s, "error:", res.err)
	}
	model := chatui.InitialModel(res.userPid, res.nodePid, a.engine)
	p := tea.NewProgram(model, tea.WithOutput(s), tea.WithInput(s))

	// now we can register the program with the user actor
	a.engine.Send(res.userPid, UserRegisterProgram{program: p})
	// and we're done.
	return p
}
