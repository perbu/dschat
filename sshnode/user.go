package sshnode

import (
	"fmt"
	"github.com/anthdm/hollywood/actor"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/perbu/dschat/msg"
)

type SshUser struct {
	Username string
	pid      *actor.PID
	engine   *actor.Engine
	program  *tea.Program
}

func (u *SshUser) Receive(c *actor.Context) {
	switch c.Message().(type) {
	case actor.Initialized:
		u.pid = c.PID()
		u.engine = c.Engine()
	case actor.Started:
	case actor.Stopped:
	case msg.Message:
		m := c.Message().(msg.Message)
		if u.program == nil {
			fmt.Println("USER: No program registered, dropping message")
			return
		}
		fmt.Println("USER: Got message:", m)
		u.program.Send(m)
	case UserRegisterProgram:
		m := c.Message().(UserRegisterProgram)
		u.program = m.program
		fmt.Println("USER: Program registered")
	default:
		fmt.Printf("USER: Unknown message(%T): %v\n", c.Message(), c.Message())
	}
}

func newUser(name string) actor.Producer {
	return func() actor.Receiver {
		return &SshUser{Username: name}
	}
}
