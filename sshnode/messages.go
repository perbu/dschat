package sshnode

import (
	"github.com/anthdm/hollywood/actor"
	tea "github.com/charmbracelet/bubbletea"
)

type NewUserRequest struct {
	userid string
}
type NewUserResponse struct {
	err     error
	nodePid *actor.PID
	userPid *actor.PID
}

type UserDisconnected struct {
	pid *actor.PID
}

type UserRegisterProgram struct {
	program *tea.Program
}
