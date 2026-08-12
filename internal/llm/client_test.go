package llm

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAvailable(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		model     string
		installed []string
		wantErr   error
	}{
		{name: "installed", model: "qwen2.5:0.5b", installed: []string{"qwen2.5:0.5b"}},
		// Тег по умолчанию: конфиг без ":" и хранилище с ":latest" — одна модель.
		{name: "default tag in config", model: "qwen2.5", installed: []string{"qwen2.5:latest"}},
		{name: "default tag in registry", model: "qwen2.5:latest", installed: []string{"qwen2.5"}},
		{
			name:      "another model pulled",
			model:     "qwen2.5:0.5b",
			installed: []string{"llama3.2:1b"},
			wantErr:   ErrModelMissing,
		},
		{name: "nothing pulled", model: "qwen2.5:0.5b", wantErr: ErrModelMissing},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			client := newClient(t, testCase.model, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/tags" {
					t.Errorf("path = %q, want /api/tags", r.URL.Path)
				}
				writeTags(t, w, testCase.installed)
			})

			err := client.Available(context.Background())
			if !errors.Is(err, testCase.wantErr) {
				t.Fatalf("Available() error = %v, want %v", err, testCase.wantErr)
			}
		})
	}
}

func TestAvailableTreatsBrokenServerAsUnavailable(t *testing.T) {
	t.Parallel()

	client := newClient(t, "qwen2.5:0.5b", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	if err := client.Available(context.Background()); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Available() error = %v, want %v", err, ErrUnavailable)
	}
}

func TestAvailableTreatsDeadServerAsUnavailable(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.NotFoundHandler())
	client := New(server.URL, "qwen2.5:0.5b")
	server.Close()

	if err := client.Available(context.Background()); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Available() error = %v, want %v", err, ErrUnavailable)
	}
}

func TestGenerateSendsSchemaAndReturnsContent(t *testing.T) {
	t.Parallel()

	schema := json.RawMessage(`{"type":"object","properties":{"scam":{"type":"boolean"}}}`)

	var sent chatRequest
	client := newClient(t, "qwen2.5:0.5b", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Errorf("path = %q, want /api/chat", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&sent); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		writeChat(t, w, `{"scam":true}`)
	})

	content, err := client.Generate(context.Background(), "ты классификатор", "перейдём в тг", schema)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if content != `{"scam":true}` {
		t.Fatalf("content = %q, want the assistant message", content)
	}

	if sent.Model != "qwen2.5:0.5b" {
		t.Errorf("model = %q, want qwen2.5:0.5b", sent.Model)
	}
	// Поток сломал бы разбор ответа: тело пришло бы построчными кусками.
	if sent.Stream {
		t.Error("stream = true, want false")
	}
	if string(sent.Format) != string(schema) {
		t.Errorf("format = %s, want the schema as given", sent.Format)
	}
	// Ноль обязателен: классификация должна повторяться, а не разнообразиться.
	if sent.Options.Temperature != 0 {
		t.Errorf("temperature = %v, want 0", sent.Options.Temperature)
	}
	if sent.Options.NumPredict != maxResponseTokens {
		t.Errorf("num_predict = %d, want %d", sent.Options.NumPredict, maxResponseTokens)
	}

	want := []chatMessage{
		{Role: "system", Content: "ты классификатор"},
		{Role: "user", Content: "перейдём в тг"},
	}
	if len(sent.Messages) != len(want) {
		t.Fatalf("messages = %v, want %v", sent.Messages, want)
	}
	for index, message := range want {
		if sent.Messages[index] != message {
			t.Errorf("messages[%d] = %v, want %v", index, sent.Messages[index], message)
		}
	}
}

func TestGenerateSkipsEmptySystemPrompt(t *testing.T) {
	t.Parallel()

	var sent chatRequest
	client := newClient(t, "qwen2.5:0.5b", func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&sent); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		writeChat(t, w, "ok")
	})

	if _, err := client.Generate(context.Background(), "", "привет", nil); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if len(sent.Messages) != 1 || sent.Messages[0].Role != "user" {
		t.Fatalf("messages = %v, want only the user turn", sent.Messages)
	}
	// Пустой format не должен уезжать как "format": null — Ollama считает это схемой.
	if sent.Format != nil {
		t.Errorf("format = %s, want it omitted", sent.Format)
	}
}

// Причина отказа лежит в теле, а не в статусе: без неё «модель не скачана» и
// «кончилась память» выглядят в логе одинаково.
func TestGenerateReportsServerErrorBody(t *testing.T) {
	t.Parallel()

	client := newClient(t, "qwen2.5:0.5b", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		if _, err := w.Write([]byte(`{"error":"model not found"}`)); err != nil {
			t.Errorf("write body: %v", err)
		}
	})

	_, err := client.Generate(context.Background(), "", "привет", nil)
	if err == nil {
		t.Fatal("Generate() error = nil, want a failure")
	}
	if !strings.Contains(err.Error(), "model not found") {
		t.Fatalf("error = %v, want it to carry the server reason", err)
	}
}

func newClient(t *testing.T, model string, handler http.HandlerFunc) *Client {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	return New(server.URL, model)
}

func writeTags(t *testing.T, w http.ResponseWriter, installed []string) {
	t.Helper()

	type model struct {
		Name string `json:"name"`
	}
	payload := struct {
		Models []model `json:"models"`
	}{}
	for _, name := range installed {
		payload.Models = append(payload.Models, model{Name: name})
	}

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Errorf("encode tags: %v", err)
	}
}

func writeChat(t *testing.T, w http.ResponseWriter, content string) {
	t.Helper()

	payload := struct {
		Message chatMessage `json:"message"`
	}{Message: chatMessage{Role: "assistant", Content: content}}

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Errorf("encode chat: %v", err)
	}
}
