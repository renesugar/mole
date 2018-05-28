package xmpp

import (
	"fmt"
)

func (c *Client) SendResultPing(id, toServer string) error {
	_, err := fmt.Fprintf(c.conn, "<iq type='result' to='%s' id='%s'/>",
		xmlEscape(toServer), xmlEscape(id))
	return err
}
