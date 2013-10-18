package irc

import "errors"

type Message struct {
	Prefix     string
	Command    string
	Parameters []string
	Raw        string
	Nick       string
	User       string
	Host       string
	Trail      string
	self       string
}

func (m *Message) IsChannelMsg() bool {
	if m.Command == CMD_PRIVMSG {
		switch m.Parameters[0][0] {
		case '#', '&', '!', '+':
			return true
		}
	}
	return false
}

func (m *Message) PrivmsgRespondTo() (target string, err error) {
	if m.IsChannelMsg() {
		target = m.Parameters[0]
	} else if m.Command == CMD_PRIVMSG {
		target = m.Nick
	} else {
		err = errors.New("Command not PRIVMSG")
	}
	return
}
