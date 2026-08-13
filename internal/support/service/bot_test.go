package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/google/uuid"

	supportmodel "github.com/sweetlife999/chain-of-trades-avito/internal/support/model"
)

type fakeGenerator struct {
	answer string
	err    error
	system string
	user   string
	calls  int
}

func (g *fakeGenerator) Generate(
	_ context.Context,
	system string,
	user string,
	_ json.RawMessage,
) (string, error) {
	g.calls++
	g.system, g.user = system, user
	return g.answer, g.err
}

type fakeBotRepository struct {
	err       error
	body      string
	calls     int
	escalated int
}

func (r *fakeBotRepository) Escalate(_ context.Context, _ uuid.UUID) error {
	r.escalated++
	return nil
}

func (r *fakeBotRepository) CreateBotMessage(
	_ context.Context,
	_ uuid.UUID,
	_ string,
	body string,
) (supportmodel.Message, error) {
	r.calls++
	r.body = body
	return supportmodel.Message{}, r.err
}

type silentLogger struct{}

func (silentLogger) Printf(string, ...any) {}

// Схема — единственное, что заставляет модель отвечать из закрытого списка, поэтому
// список тем в ней и список заготовок обязаны совпадать. Разъедутся — пользователь
// получит пустое сообщение, а база отвергнет вставку.
func TestBotAnswersMatchSchema(t *testing.T) {
	t.Parallel()

	var schema struct {
		Properties struct {
			Topic struct {
				Enum []string `json:"enum"`
			} `json:"topic"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(botSchema, &schema); err != nil {
		t.Fatalf("botSchema is not valid JSON: %v", err)
	}

	if len(schema.Properties.Topic.Enum) != len(botAnswers) {
		t.Fatalf(
			"schema has %d topics, botAnswers has %d",
			len(schema.Properties.Topic.Enum),
			len(botAnswers),
		)
	}
	for _, topic := range schema.Properties.Topic.Enum {
		answer, ok := botAnswers[topic]
		if !ok {
			t.Errorf("topic %q has no answer", topic)
			continue
		}
		if strings.TrimSpace(answer) == "" {
			t.Errorf("topic %q has an empty answer", topic)
		}
		// CHECK в support_messages ограничивает тело двумя тысячами символов.
		if length := utf8.RuneCountInString(botDisclaimer + answer); length > maxMessageLength {
			t.Errorf("answer for %q is %d characters, database allows %d", topic, length, maxMessageLength)
		}
	}
}

func TestBotReply(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name          string
		answer        string
		generateErr   error
		repositoryErr error
		wantErr       bool
		wantSaved     bool
		wantContains  string
		// Эскалация — «обращению нужен человек». Ставится, когда бот не справился:
		// не смог ответить вовсе или отдал общий текст темы other.
		wantEscalated bool
	}{
		{
			name:         "known topic",
			answer:       `{"topic":"delivery"}`,
			wantSaved:    true,
			wantContains: "пункты выдачи",
		},
		{
			// Схема закрывает список, но обновление сервера или модели может принести
			// чужое значение. Общий ответ безопаснее молчания и не врёт пользователю.
			name:          "unknown topic falls back to other",
			answer:        `{"topic":"refund"}`,
			wantSaved:     true,
			wantContains:  botAnswers[topicOther],
			wantEscalated: true,
		},
		{
			// Модель честно вернула other: ответа по существу не было, человек нужен
			// сразу, а не после того, как пользователь напишет второй раз.
			name:          "explicit other escalates",
			answer:        `{"topic":"other"}`,
			wantSaved:     true,
			wantContains:  botAnswers[topicOther],
			wantEscalated: true,
		},
		{
			name:          "broken json",
			answer:        "topic: delivery",
			wantErr:       true,
			wantSaved:     false,
			wantEscalated: true,
		},
		{
			// Сбой модели не должен ломать обращение: оно уже создано и лежит в очереди.
			name:          "model unavailable",
			answer:        "",
			generateErr:   errors.New("ollama is unavailable"),
			wantErr:       true,
			wantSaved:     false,
			wantEscalated: true,
		},
		{
			// Обращение успели закрыть или взять в работу — штатный исход, не ошибка.
			name:          "thread no longer waits",
			answer:        `{"topic":"account"}`,
			repositoryErr: ErrConflict,
			wantSaved:     true,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			model := &fakeGenerator{answer: testCase.answer, err: testCase.generateErr}
			repository := &fakeBotRepository{err: testCase.repositoryErr}
			bot := newBot(repository, model, silentLogger{})

			err := bot.reply(context.Background(), botJob{threadID: uuid.New(), text: "тема\nтекст"})

			if testCase.wantErr && err == nil {
				t.Fatal("expected an error, got nil")
			}
			if !testCase.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if saved := repository.calls == 1; saved != testCase.wantSaved {
				t.Fatalf("message saved = %v, want %v", saved, testCase.wantSaved)
			}
			if escalated := repository.escalated == 1; escalated != testCase.wantEscalated {
				t.Fatalf("thread escalated = %v, want %v", escalated, testCase.wantEscalated)
			}
			if !testCase.wantSaved {
				return
			}
			if !strings.HasPrefix(repository.body, botDisclaimer) {
				t.Error("saved message does not warn that the answer is automatic")
			}
			if testCase.wantContains != "" && !strings.Contains(repository.body, testCase.wantContains) {
				t.Errorf("saved message does not contain %q", testCase.wantContains)
			}
		})
	}
}

// Очередь и воркер: задача доходит от Enqueue до вставки, а закрытая очередь
// останавливает Run, дочитав буфер.
func TestBotQueueDeliversJob(t *testing.T) {
	t.Parallel()

	model := &fakeGenerator{answer: `{"topic":"complaint"}`}
	repository := &fakeBotRepository{}
	bot := newBot(repository, model, silentLogger{})

	bot.Enqueue(uuid.New(), "Грубость", "Участник оскорбляет меня в чате обмена")
	bot.Close()
	bot.Run(context.Background())

	if repository.calls != 1 {
		t.Fatalf("repository calls = %d, want 1", repository.calls)
	}
	if !strings.Contains(repository.body, "Пожаловаться") {
		t.Error("complaint answer does not mention the report button")
	}
	if !strings.Contains(model.user, "Грубость") || !strings.Contains(model.user, "оскорбляет") {
		t.Errorf("model got %q, want both subject and body", model.user)
	}
}

// Пустой OLLAMA_URL отдаёт нулевого бота, и поддержка обязана работать как раньше.
func TestNilBotIsDisabledFeature(t *testing.T) {
	t.Parallel()

	var bot *Bot

	bot.Enqueue(uuid.New(), "тема", "текст")
	bot.Close()
	bot.Run(context.Background())
}

func TestBotInputTruncatesByRunes(t *testing.T) {
	t.Parallel()

	long := strings.Repeat("я", botInputLimit*2)
	got := botInput("тема", long)

	if length := utf8.RuneCountInString(got); length != botInputLimit {
		t.Fatalf("input length = %d runes, want %d", length, botInputLimit)
	}
	if !utf8.ValidString(got) {
		t.Fatal("truncation broke a multibyte character")
	}
	if short := botInput("тема", "короткий текст"); short != "тема\nкороткий текст" {
		t.Fatalf("short input changed: %q", short)
	}
}
