package msg

import "github.com/anthdm/hollywood/actor"

// We use two different messages for ingress and egress messages, so that we can
// differentiate between messages that are sent from the node to the user, and
// messages that are sent from the user to the node.
// This helps avoid loops.

type IngressMessage struct {
	From *actor.PID
	To   string
	Msg  string
}

type EgressMessage struct {
	From *actor.PID
	To   string
	Msg  string
}

func (m EgressMessage) ToIngress() IngressMessage {
	return IngressMessage{
		From: m.From,
		To:   m.To,
		Msg:  m.Msg,
	}
}

type UserDisconnected struct {
}
