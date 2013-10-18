package irc

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"net"
	"strings"
	"time"
)

type Client struct {
	conn      net.Conn
	out       chan string
	in        chan string
	done      chan bool
	stopPing  chan bool
	callbacks map[string][]func(*Message)
	server    string
	nick      string
	ssl       bool
}

func (c *Client) Connect() error {
	var err error
	if c.ssl {
		c.conn, err = tls.Dial("tcp", c.server, nil)
	} else {
		c.conn, err = net.Dial("tcp", c.server)
	}

	if err != nil {
		return err
	}

	c.openChannels()
	c.registerCallbacks()

	go c.startDebugLoop()
	go c.startReadLoop()
	go c.startWriteLoop()
	go c.startPingLoop()

	c.Writeln("NICK %s", c.nick)
	c.Writeln("USER %s %s %s :%s", c.nick, "8", "*", c.nick)
	return err
}

func (c *Client) Disconnect() {
	c.Writeln("QUIT")
	c.stopPing <- true
	c.done <- true
}

func (c *Client) Wait() {
	<-c.done
	fmt.Printf("Cleaning up...")
	close(c.in)
	close(c.out)
	c.conn.Close()
	fmt.Println("done.")
}

func (c *Client) Join(channel string) {
	c.Writeln("JOIN %s", channel)
}

func (c *Client) Part(channel string) {
	c.Writeln("PART %s", channel)
}

func (c *Client) Privmsg(target, message string) {
	c.Writeln("PRIVMSG %s :%s", target, message)
}

func (c *Client) ChangeNick(nick string) {
	c.nick = nick
	c.Writeln("NICK %s", nick)
}

func (c *Client) GetNick() string {
	return c.nick
}

func (c *Client) Writeln(format string, vars ...interface{}) {
	c.in <- fmt.Sprintf(format, vars...) + "\r\n"
}

func (c *Client) DebugWrite(format string, vars ...interface{}) {
	c.out <- fmt.Sprintf(format, vars...)
}

func (c *Client) HandleCommand(cmd string, callback func(*Message)) {
	if _, exists := c.callbacks[cmd]; exists {
		c.callbacks[cmd] = append(c.callbacks[cmd], callback)
	} else {
		c.callbacks[cmd] = make([]func(*Message), 1)
		c.callbacks[cmd][0] = callback
	}
}

func (c *Client) openChannels() {
	c.in = make(chan string)
	c.out = make(chan string)
	c.done = make(chan bool)
	c.stopPing = make(chan bool)
}

func (c *Client) registerCallbacks() {
	// some servers ping the client on connect, lets handle that
	c.HandleCommand("PING", func(msg *Message) {
		c.Writeln("PONG %s", msg.Trail)
	})

	// whoops, nick is in use. change it.
	c.HandleCommand(ERR_NICKINUSE, func(msg *Message) {
		newnick := c.nick + "-"
		c.DebugWrite("Nick %s is in use, changing to %s.", c.nick, newnick)
		c.ChangeNick(newnick)
	})
}

func (c *Client) startDebugLoop() {
	for {
		if msg, ok := <-c.out; !ok {
			break
		} else {
			fmt.Println(strings.TrimRight(msg, "\n"))
		}
	}
	c.done <- true
}

func (c *Client) startReadLoop() {
	r := bufio.NewReaderSize(c.conn, 512)

	for {
		if line, err := r.ReadString('\n'); err != nil {
			break
		} else {
			msg := c.parseMessage(strings.TrimRight(line, "\r\n"))
			if funcs, exists := c.callbacks[msg.Command]; !exists {
				funcs = c.callbacks["*"]
			}
			for _, cb := range funcs {
				go cb(msg)
			}
		}
	}
	c.done <- true
}

func (c *Client) startWriteLoop() {
	for {
		if msg, ok := <-c.in; !ok {
			break
		} else {
			if _, e := c.conn.Write([]byte(msg)); e != nil {
				break
			}
		}
	}

	c.done <- true
}

func (c *Client) startPingLoop() {
	interval := time.NewTicker(90 * time.Second)
	for {
		select {
		case <-interval.C:
			c.Writeln("PING %d", time.Now().UnixNano())
		case <-c.stopPing:
			fmt.Println("Stopping ping loop.")
			break
		}
	}
}

func (c *Client) parseMessage(raw string) *Message {
	result := &Message{Raw: raw, self: c.nick}
	hasPrefix := raw[0] == ':'

	var prefixIndex int
	if hasPrefix {
		prefixIndex = strings.Index(raw, " ")
	} else {
		prefixIndex = -1
	}

	params := strings.SplitN(raw[prefixIndex+1:], " :", 2)
	cmd := strings.Split(params[0], " ")

	if hasPrefix {
		result.Prefix = raw[1:prefixIndex]
	}
	result.Command = strings.ToUpper(cmd[0])
	result.Parameters = cmd[1:]
	if len(params) > 1 {
		result.Trail = params[1]
	}

	split := strings.FieldsFunc(result.Prefix, func(r rune) bool { return r == '!' || r == '@' })
	if len(split) == 3 {
		result.Nick = split[0]
		result.User = split[1]
		result.Host = split[2]
	}
	return result
}
