package tui

import (
	"context"

	tea "charm.land/bubbletea/v2"
	"github.com/cheetahbyte/apex/internal/conversation"
	"github.com/cheetahbyte/apex/internal/llm"
	"github.com/cheetahbyte/apex/internal/tui/components/chat"
	"github.com/cheetahbyte/apex/internal/tui/components/prompt"
	"github.com/cheetahbyte/apex/internal/tui/components/sidebar"
	"github.com/cheetahbyte/apex/internal/tui/components/statusline"
)

type (
	streamChunkMsg string
	streamDoneMsg  struct{}
	errMsg         struct{ err error }
)

// Model is the root Bubble Tea model. It owns only layout state and
// delegates rendering and event handling to child components. LLM
// communication goes through the llm.Client interface and chat history
// lives in conversation.Session.
type Model struct {
	width  int
	height int

	chat    chat.Model
	prompt  prompt.Model
	sidebar sidebar.Model
	status  statusline.Model

	session   *conversation.Session
	streaming bool
	chunks    chan tea.Msg
	client    llm.Client
}

// New creates the root TUI model. The LLM client is injected so the TUI
// stays decoupled from any specific provider.
func New(client llm.Client) Model {
	return Model{
		chat:     chat.New(),
		prompt:   prompt.New(),
		sidebar:  sidebar.New(),
		status:   statusline.New(),
		session:  conversation.NewSession(),
		client:   client,
		chunks:   make(chan tea.Msg),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(prompt.Blink())
}

// spawnStream starts an LLM stream for the current session and forwards
// events into the chunks channel so Bubble Tea can process them as messages.
func (m Model) spawnStream() tea.Cmd {
	return func() tea.Msg {
		events := m.client.Stream(context.Background(), m.session.Messages())
		go func() {
			for ev := range events {
				switch {
				case ev.Err != nil:
					m.chunks <- errMsg{ev.Err}
					return
				case ev.Done:
					m.chunks <- streamDoneMsg{}
					return
				default:
					m.chunks <- streamChunkMsg(ev.Delta)
				}
			}
		}()
		return nil
	}
}

func waitForChunk(ch chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		return <-ch
	}
}
