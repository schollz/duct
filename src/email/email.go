package email

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"
	"github.com/pkg/errors"
	log "github.com/schollz/logger"
)

// Email is the main structure with the email
type Email struct {
	To      string    `json:"to"`
	From    string    `json:"from"`
	Date    time.Time `json:"date"`
	Subject string    `json:"subject"`
	Body    string    `json:"body"`
}

func (e Email) String() string {
	bJSON, _ := json.Marshal(e)
	return string(bJSON)
}

// Send uses the specified credential file to send an email
func Send(flagEmailCreds string, e Email) (err error) {

	return
}

// GetLatest returns the latest email
func GetLatest(flagEmailCreds string) (recvEmail Email, err error) {
	log.Debug("Connecting to server...")

	// Connect to server
	c, err := client.DialTLS("imap.gmail.com:993", nil)
	if err != nil {
		log.Error(err)
		return
	}
	log.Debug("Connected")

	// Don't forget to logout
	defer c.Logout()

	// Login
	b, err := ioutil.ReadFile(flagEmailCreds)
	if err != nil {
		log.Error(err)
		return
	}
	fields := strings.Fields(string(b))
	if err = c.Login(fields[0], fields[1]); err != nil {
		log.Error(err)
		return
	}
	log.Debug("Logged in")

	// Select INBOX
	mbox, err := c.Select("INBOX", false)
	if err != nil {
		log.Error(err)
		return
	}

	// Get the last message
	if mbox.Messages == 0 {
		err = fmt.Errorf("No message in mailbox")
		log.Debug(err)
		return
	}
	seqSet := new(imap.SeqSet)
	log.Debugf("found %d messages", mbox.Messages)
	// seqSet.AddNum(mbox.Messages)
	seqSet.AddRange(1, mbox.Messages)

	// Get the whole message body
	var section imap.BodySectionName
	items := []imap.FetchItem{section.FetchItem()}

	messages := make(chan *imap.Message, mbox.Messages)
	go func() {
		if err := c.Fetch(seqSet, items, messages); err != nil {
			log.Error(err)
		}
	}()
	for {
		msg := <-messages
		if msg == nil {
			log.Error("Server didn't returned message")
		}

		r := msg.GetBody(&section)
		if r == nil {
			log.Error("Server didn't returned message body")
		}

		// Create a new mail reader
		var mr *mail.Reader
		mr, err = mail.CreateReader(r)
		if err != nil {
			log.Error(err)
			return
		}

		// Print some info about the message
		header := mr.Header
		recvEmail.Date, err = header.Date()
		var ad []*mail.Address
		ad, err = header.AddressList("From")
		recvEmail.From = fmt.Sprintf("%s", ad[0].Address)
		ad, err = header.AddressList("To")
		recvEmail.To = fmt.Sprintf("%s", ad[0].Address)
		recvEmail.Subject, err = header.Subject()

		// Process each message's part
		for {
			p, err := mr.NextPart()
			if err == io.EOF {
				break
			} else if err != nil {
				log.Error(err)
			}

			switch h := p.Header.(type) {
			case *mail.InlineHeader:
				// This is the message's text (can be plain-text or HTML)
				log.Debug("a new part")
				b, _ := ioutil.ReadAll(p.Body)
				log.Debugf("Got text: '%v'", string(b))
				if recvEmail.Body == "" {
					recvEmail.Body = strings.TrimSpace(string(b))
				}
			case *mail.AttachmentHeader:
				// This is an attachment
				filename, _ := h.Filename()
				log.Debugf("Got attachment: %v", filename)
				b, _ := ioutil.ReadAll(p.Body)
				log.Debugf("Got attachment: %d bytes", len(b))
			}
		}

		// delete it
		flags := []interface{}{imap.DeletedFlag}
		if err = c.Store(seqSet, imap.AddFlags, flags, nil); err != nil {
			log.Error(err)
			return
		}

		if err = c.Expunge(nil); err != nil {
			err = errors.Wrap(err, "IMAP Message Delete Failed")
			return
		}
		break
	}
	return
}
