package conversation

// Role identifies who produced a message in the conversation.
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

// Message is a single chat message with a role and content.
type Message struct {
	Role    Role
	Content string
}

// Session tracks the full conversation history sent to and received from
// the LLM. It is the single source of truth for chat state. The TUI
// renders from it rather than maintaining its own parallel list.
type Session struct {
	messages []Message
}

// NewSession returns an empty conversation.
func NewSession() *Session {
	return &Session{}
}

// AppendUser adds a user message to the history.
func (s *Session) AppendUser(content string) {
	s.messages = append(s.messages, Message{Role: RoleUser, Content: content})
}

// AppendAssistant adds an assistant response to the history.
func (s *Session) AppendAssistant(content string) {
	s.messages = append(s.messages, Message{Role: RoleAssistant, Content: content})
}

// AppendSystem adds a system prompt to the history.
func (s *Session) AppendSystem(content string) {
	s.messages = append(s.messages, Message{Role: RoleSystem, Content: content})
}

// Messages returns the full message list for sending to an LLM provider.
func (s *Session) Messages() []Message {
	return s.messages
}

// Len returns the number of messages in the session.
func (s *Session) Len() int {
	return len(s.messages)
}
