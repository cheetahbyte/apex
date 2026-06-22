package tui

import (
	"context"
	"strings"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"github.com/cheetahbyte/apex/internal/tui/sidebar"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

type (
	streamChunkMsg string
	streamDoneMsg  struct{}
	errMsg         struct{ err error }
)

type Model struct {
	width  int
	height int

	chat      viewport.Model
	input     textinput.Model
	sidebar   sidebar.Model
	messages  []string
	output    string
	streaming bool
	chunks    chan tea.Msg
	client    openai.Client
}

func New() Model {
	input := textinput.New()
	input.Placeholder = "Lets work!"
	input.Focus()
	input.CharLimit = 4000
	input.Prompt = "> "

	messages := []string{"Apex ready."}
	output := strings.Join(messages, "\n\n")
	chat := viewport.New()
	chat.SetContent(output)

	sidebar := sidebar.New()

	client := openai.NewClient(
		option.WithBaseURL("http://localhost:11434/v1"),
		option.WithAPIKey("ollama"),
	)

	return Model{
		chat:     chat,
		input:    input,
		sidebar:  sidebar,
		messages: messages,
		client:   client,
		output:   output,
		chunks:   make(chan tea.Msg),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink)
}

func (m Model) spawnStream(prompt string) tea.Cmd {
	return func() tea.Msg {
		go func() {
			stream := m.client.Chat.Completions.NewStreaming(context.Background(),
				openai.ChatCompletionNewParams{
					Model:    "llama3.2",
					Messages: []openai.ChatCompletionMessageParamUnion{openai.UserMessage(prompt)},
				})
			for stream.Next() {
				if c := stream.Current().Choices; len(c) > 0 {
					m.chunks <- streamChunkMsg(c[0].Delta.Content)
				}
			}
			if err := stream.Err(); err != nil {
				m.chunks <- errMsg{err}
			}
			m.chunks <- streamDoneMsg{}
		}()
		return nil
	}
}

func waitForChunk(ch chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		return <-ch
	}
}
