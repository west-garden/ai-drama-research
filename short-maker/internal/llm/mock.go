package llm

import "context"

type MockClient struct {
	responses map[string]string // model -> canned response
	calls     []Request
}

func NewMockClient() *MockClient {
	return &MockClient{
		responses: map[string]string{},
	}
}

func (m *MockClient) SetResponse(model, response string) {
	m.responses[model] = response
}

func (m *MockClient) SetDefaultResponse(response string) {
	m.responses["*"] = response
}

func (m *MockClient) Chat(ctx context.Context, req Request) (*Response, error) {
	m.calls = append(m.calls, req)
	content, ok := m.responses[req.Model]
	if !ok {
		content = m.responses["*"]
	}
	return &Response{
		Content:    content,
		TokensUsed: len(content),
		Model:      req.Model,
	}, nil
}

func (m *MockClient) Calls() []Request { return m.calls }
