package irc

const (
	RPL_WELCOME      string = "001"
	RPL_TOPIC        string = "332"
	RPL_TOPICWHOTIME string = "333"
	RPL_NAMEREPLY    string = "353"
	RPL_ENDOFNAMES   string = "366"
	RPL_MOTDSTART    string = "375"
	RPL_MOTD         string = "372"
	RPL_ENDOFMOTD    string = "376"

	CMD_NOTICE  string = "NOTICE"
	CMD_PRIVMSG string = "PRIVMSG"
	CMD_JOIN    string = "JOIN"
	CMD_KICK    string = "KICK"
	CMD_MODE    string = "MODE"

	ERR_NOMOTD    string = "422"
	ERR_NICKINUSE string = "433"
)

func NewClient(server, nick string, ssl bool) *Client {
	client := &Client{
		server:    server,
		nick:      nick,
		ssl:       ssl,
		callbacks: make(map[string][]func(*Message)),
	}
	return client
}
