package sshnode

import (
	"fmt"
	"github.com/anthdm/hollywood/actor"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/perbu/dschat/msg"
)

type SshUser struct {
	Username string
	userPid  *actor.PID
	engine   *actor.Engine
	//	nodePid  *actor.PID // just use the parent pid
	program *tea.Program
}

func (u *SshUser) Receive(c *actor.Context) {
	switch c.Message().(type) {
	case actor.Initialized:
		u.userPid = c.PID()
		u.engine = c.Engine()
	case actor.Started:
	case actor.Stopped:
	case msg.IngressMessage:
		m := c.Message().(msg.IngressMessage)
		u.program.Send(m) // send to the program, so it is displayed
	case msg.EgressMessage:
		m := c.Message().(msg.EgressMessage)

		if u.program == nil {
			fmt.Println("USER: No program registered, dropping message")
			return
		}
		// Send it up to the node, if it didn't come from the node or from ourself:
		fmt.Println("USER: Sending message to node")
		u.engine.Send(c.Parent(), m)
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
