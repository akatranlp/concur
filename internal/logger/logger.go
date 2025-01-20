package logger

import (
	"os"

	"github.com/akatranlp/concur/internal/prefix"
)

type Message struct {
	ID   int
	Text string
}

type PrefixLogger struct {
	prefix *prefix.Prefix
	out    *os.File

	done  chan struct{}
	msgCh chan Message
}

func NewPrefixLogger(p *prefix.Prefix, output *os.File) *PrefixLogger {
	return &PrefixLogger{
		prefix: p,
		out:    output,
		msgCh:  make(chan Message, 100),
		done:   make(chan struct{}),
	}
}

func (l *PrefixLogger) GetMessageChannel() chan<- Message {
	return l.msgCh
}

func (l *PrefixLogger) Close() {
	close(l.msgCh)
}

func (l *PrefixLogger) Run() {
	defer close(l.done)
	for msg := range l.msgCh {
		prefix := l.prefix.Render(msg.ID, true)
		l.out.WriteString(prefix)
		l.out.WriteString(msg.Text)
	}
}

func (l *PrefixLogger) Wait() {
	<-l.done
}
