package msg

import "github.com/anthdm/hollywood/actor"

type Message struct {
	From *actor.PID
	To   *actor.PID
	Msg  string
}
