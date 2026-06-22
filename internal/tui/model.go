package tui

import (
	"context"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"github.com/cheetahbyte/apex/internal/conversation"
	"github.com/cheetahbyte/apex/internal/llm"
	"github.com/cheetahbyte/apex/internal/tui/components/sidebar"
)

type (
	streamChunkMsg string
	streamDoneMsg  struct{}
	errMsg         struct{ err error }
)

// Model is the root Bubble Tea model. It owns only UI state and delegates
// LLM communication to the llm.Client interface and chat history to the
// conversation.Session.
type Model struct {
	width  int
	height int

	chat    viewport.Model
	input   textinput.Model
	sidebar sidebar.Model

	session       *conversation.Session
	output        string // rendered chat text for the viewport
	assistantBuf  string // accumulates the current assistant response
	streaming     bool
	chunks        chan tea.Msg
	client        llm.Client
}

// New creates the root TUI model. The LLM client is injected so the TUI
// stays decoupled from any specific provider.
func New(client llm.Client) Model {
	input := textinput.New()
	input.Placeholder = "Lets work!"
	input.Focus()
	input.CharLimit = 4000
	input.Prompt = "> "

	output := "Apex ready."
	chat := viewport.New()
	chat.SetContent(output)

	sidebar := sidebar.New()
	session := conversation.NewSession()

	return Model{
		chat:     chat,
		input:    input,
		sidebar:  sidebar,
		session:  session,
		client:   client,
		output:   output,
		chunks:   make(chan tea.Msg),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink)
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
