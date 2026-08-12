// Package llm обращается к локальной модели, которую поднимает Ollama соседним
// контейнером. Пакет знает ровно одну операцию — короткий вопрос с ответом по
// JSON-схеме: и антискам в чате обмена, и маршрутизация обращений в поддержку
// сводятся к ней, поэтому второго метода генерации не появляется.
//
// Внешней библиотеки под Ollama нет намеренно: это один GET и один POST, а
// зависимость пришлось бы обновлять и объяснять. Подробности — docs/llm.md.
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var (
	// ErrUnavailable — до сервера Ollama не достучались.
	ErrUnavailable = errors.New("ollama is unavailable")
	// ErrModelMissing — сервер отвечает, но модель на нём не скачана. Лечится
	// иначе, чем ErrUnavailable, поэтому это отдельная ошибка, а не одна на двоих.
	ErrModelMissing = errors.New("ollama model is not pulled")
)

const (
	// Верхняя граница на случай, если Ollama примет соединение и замолчит. Реальный
	// срок задаёт context вызывающего: на одном ядре разброс между быстрым и
	// медленным ответом слишком велик, чтобы зашивать его константой.
	requestTimeout = 2 * time.Minute
	// Ответ классификатора — несколько токенов. Более длинный означает, что модель
	// ушла не туда, и оборвать его дешевле, чем дочитать.
	maxResponseTokens = 256
	// Тело ошибки нужно целиком только в патологическом случае, а в лог оно едет
	// одной строкой.
	maxErrorBytes = 512
)

type Client struct {
	baseURL string
	model   string
	http    *http.Client
}

func New(baseURL, model string) *Client {
	return &Client{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		model:   model,
		http:    &http.Client{Timeout: requestTimeout},
	}
}

// Available проверяет, что сервер поднят и нужная модель уже скачана. Инференс при
// этом не запускается намеренно: на одноядерном сервере прогрев весов ради проверки
// живости занял бы ровно то ядро, на котором отвечает API.
func (c *Client) Available(ctx context.Context) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/tags", nil)
	if err != nil {
		return fmt.Errorf("build tags request: %w", err)
	}

	response, err := c.http.Do(request)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrUnavailable, err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: tags returned %s", ErrUnavailable, response.Status)
	}

	var payload struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return fmt.Errorf("decode tags response: %w", err)
	}

	wanted := withDefaultTag(c.model)
	for _, model := range payload.Models {
		if withDefaultTag(model.Name) == wanted {
			return nil
		}
	}

	return fmt.Errorf("%w: %s", ErrModelMissing, c.model)
}

// Generate задаёт модели один вопрос и возвращает ответ как есть. Непустой format —
// это JSON-схема, и тогда Ollama сам принуждает ответ к ней, так что разбирать прозу
// вызывающему не приходится. Ответ остаётся строкой, а не json.RawMessage: без схемы
// в нём лежит обычный текст, и выдавать его за JSON было бы враньём в типе.
//
// TODO: продакшн-вызывающего пока нет, первым станет антискам-воркер в чате обмена.
func (c *Client) Generate(
	ctx context.Context,
	system string,
	user string,
	format json.RawMessage,
) (string, error) {
	body := chatRequest{
		Model: c.model,
		// Ответ нужен целиком и сразу: он короткий, а поток заставил бы склеивать
		// куски ради пары десятков токенов.
		Stream: false,
		Format: format,
		Options: chatOptions{
			// Классификации нужен воспроизводимый ответ, а не разнообразие.
			Temperature: 0,
			NumPredict:  maxResponseTokens,
		},
	}
	if system != "" {
		body.Messages = append(body.Messages, chatMessage{Role: "system", Content: system})
	}
	body.Messages = append(body.Messages, chatMessage{Role: "user", Content: user})

	encoded, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("encode chat request: %w", err)
	}

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/api/chat",
		bytes.NewReader(encoded),
	)
	if err != nil {
		return "", fmt.Errorf("build chat request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := c.http.Do(request)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrUnavailable, err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("chat returned %s: %s", response.Status, errorBody(response.Body))
	}

	var payload struct {
		Message chatMessage `json:"message"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("decode chat response: %w", err)
	}

	return payload.Message.Content, nil
}

type chatRequest struct {
	Model    string          `json:"model"`
	Messages []chatMessage   `json:"messages"`
	Stream   bool            `json:"stream"`
	Format   json.RawMessage `json:"format,omitempty"`
	Options  chatOptions     `json:"options"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatOptions struct {
	Temperature float64 `json:"temperature"`
	NumPredict  int     `json:"num_predict"`
}

// Ollama хранит модель без тега как «:latest», поэтому «qwen2.5» и «qwen2.5:latest»
// — одно имя. Без приведения проверка доступности врала бы про отсутствие модели.
func withDefaultTag(model string) string {
	if strings.Contains(model, ":") {
		return model
	}

	return model + ":latest"
}

// Ollama кладёт причину в {"error": "..."}, но при падении внутри сервера отдаёт и
// обычный текст. Поэтому берём тело как есть, ограничив длину.
func errorBody(body io.Reader) string {
	snippet, err := io.ReadAll(io.LimitReader(body, maxErrorBytes))
	if err != nil {
		return "<unreadable>"
	}

	return strings.TrimSpace(string(snippet))
}
