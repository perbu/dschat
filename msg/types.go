package msg

import "github.com/anthdm/hollywood/actor"

type Message struct {
	From *actor.PID
	To   string // who is the message to, by user id
	Msg  string
}
