package xmpp

import (
	"crypto/tls"

	"github.com/frankbraun/mole/config"
	"github.com/frankbraun/mole/util"
	"github.com/frankbraun/mole/util/log"
	"github.com/mattn/go-xmpp"
)

// Start XMPP client for the given account.
// Messages are from send channel and sent to server.
// Messages retrieved from server a written to recv channel.
func Start(
	account *config.Account,
	send <-chan string,
	recv chan<- string,
	debug bool,
) error {
	xmpp.DefaultConfig = tls.Config{
		InsecureSkipVerify: true,
	}
	options := xmpp.Options{
		User:     account.Username,
		Password: account.Password,
		NoTLS:    true,
		StartTLS: true,
		Debug:    debug,
	}

	talk, err := options.NewClient()
	if err != nil {
		return err
	}

	go func() {
		for {
			chat, err := talk.Recv()
			if err != nil {
				// TODO: better handling
				log.Printf("fatal(): %v", err)
				util.Fatal(err)
			}
			switch v := chat.(type) {
			case xmpp.Chat:
				recv <- v.Text
			case xmpp.Presence:
				// TODO: handle presence messages
				// fmt.Println(v.From, v.Show)
			}
		}
	}()

	go func() {
		for {
			msg := <-send
			talk.Send(xmpp.Chat{Remote: account.Contact, Type: "chat", Text: msg})
		}
	}()

	return nil
}
