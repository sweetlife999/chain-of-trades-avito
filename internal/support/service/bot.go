package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	supportmodel "github.com/sweetlife999/chain-of-trades-avito/internal/support/model"
)

// BotNickname обязан совпадать с ником служебного пользователя: запрос находит его
// джойном по нику, поэтому UUID в Go не хранится. Пользователь заведён миграцией 00024 и
// переименован в 00028. Разъехавшись, эти два места дают тихое молчание бота, а не
// ошибку, — поэтому совпадение проверяется на старте (см. newSupportBot в cmd/api).
//
// Ник виден пользователю над сообщением, и это имя маскота.
const BotNickname = "Уми"

const (
	// Классификатору хватает первых фраз: тема обращения всегда в них. Ограничение
	// нужно не для качества, а для времени ответа — оно определяется длиной входа.
	// Замерено на одном потоке: 400 символов — около 1.3 с префилла, полные 2000 —
	// около 2.7 с, и это ещё без прод-квоты в половину ядра.
	botInputLimit = 400
	// Обращения появляются редко: активное обращение у пользователя может быть только
	// одно, и автоответ на него один. Переполнение буфера означает не нагрузку, а
	// сломанный воркер, поэтому очередь маленькая и задачи из неё выбрасываются.
	botQueueCapacity = 32
	// Верхняя граница на один вызов модели с запасом на холодный старт: веса
	// выгружаются после 30 с простоя (OLLAMA_KEEP_ALIVE), и обращение почти всегда
	// приходит на выгруженную модель. Замерено: загрузка с диска — около 1.7 с.
	botCallTimeout = 30 * time.Second
)

// Темы повторяют enum в botSchema. Обе задачи, которые сервис поручает модели, —
// классификация, поэтому список закрытый: свободный текст 0.5B на русском не пишет.
const (
	topicDelivery  = "delivery"
	topicExchange  = "exchange"
	topicPayment   = "payment"
	topicAccount   = "account"
	topicComplaint = "complaint"
	topicOther     = "other"
)

// Промпт по-английски при русских обращениях: замерено, что так точнее. Ни одной
// дополнительной инструкции: попытка дописать правило («не уверен — отвечай other»)
// роняла точность вдвое, с 11 из 15 до 6 из 15 — на 0.5B место в промпте кончается
// быстро, а отрицаний она не держит вовсе.
//
// Правило про примеры не «чем меньше, тем лучше», а **ровно один пример на тему**:
// восемь примеров на пять тем давали 10 из 16 против 11 у пяти примеров, а шесть
// примеров на шесть тем дают 14 из 19 против 12 у пяти. Появится седьмая тема —
// появится и седьмой пример.
//
// Слово «статус» стоит у delivery, а не у exchange: замерено, 11 из 16 против 10 у
// версии, где оно было у обмена. Правка задумывалась ради обращений про статус доставки
// и как раз их не вылечила, зато починила два посторонних кейса — связь между правкой
// промпта и результатом не прямая, поэтому правки этого текста надо мерить, а не
// обсуждать. Порог держит bot_llm_test.go.
const botPrompt = `You route support requests for an item-swap service where users exchange things in chains through pickup points. Requests are in Russian. Pick exactly one topic.
- delivery: pickup points, handing the item over, delivery status, tracking, item did not arrive, courier
- exchange: how chains are found, waiting for other participants, cancelling, reservations
- payment: money, price, fees, commission, paying for the service, extra payment between participants
- account: login, password, nickname, profile, photo, notifications settings
- complaint: another user behaved badly, fraud, offensive messages, someone asks for money
- other: anything else
Examples:
"Не могу войти, пароль не подходит" -> account
"Отдал вещь в ПВЗ, а статус не поменялся" -> delivery
"Почему цепочка отменилась сама?" -> exchange
"Сколько стоит обмен?" -> payment
"Участник мне грубит в чате обмена" -> complaint
"Хочу предложить вам идею для сервиса" -> other`

// Схема заставляет Ollama вернуть одно из перечисленных значений, поэтому разбирать
// прозу и подчищать ответ не приходится. Значения-слова, а не bool и не номера: на bool
// 0.5B залипает и отвечает одно и то же на любой вход, а на номерах ответов дала 5 из 15
// против 11 у слов. Порядок значений повторяет порядок тем в промпте: замеры делались на
// этой раскладке, а модель к ней чувствительна.
var botSchema = json.RawMessage(
	`{"type":"object","properties":{"topic":{"type":"string",` +
		`"enum":["delivery","exchange","payment","account","complaint","other"]}},` +
		`"required":["topic"]}`,
)

// Первая строка каждого ответа. Модель ошибается примерно в каждом четвёртом обращении,
// поэтому пользователь должен видеть, что ответ автоматический, и знать, что за ним
// придёт человек.
const botDisclaimer = "Привет! Я Уми. Я могу ошибаться, поэтому если проблема не будет решена, напишите, и я позову модератора.\n\n"

// Тексты написаны руками и поэтому верны. Модели их писать нельзя: с фактами в промпте
// она выдумывает функции, которых нет, и противоречит тем же фактам — на пяти
// проверочных вопросах верным был один ответ. Модель только выбирает, какой из этих
// текстов отдать.
var botAnswers = map[string]string{
	topicDelivery: "Вещи в сервисе передаются только через пункты выдачи: отнесите свою вещь в выбранный " +
		"ПВЗ, а забрать встречную можно после того, как её отметят доставленной. " +
		"Статус обмена меняется не мгновенно — его подтверждает сотрудник пункта выдачи. " +
		"Если вещь не пришла или статус не двигается дольше суток, напишите номер обмена: модератор проверит.",
	topicExchange: "Цепочку сервис собирает сам: от двух до пяти участников, каждый отдаёт свою вещь и " +
		"получает вещь другого. Пока согласились не все, обмен остаётся в статусе «предложен» — " +
		"ждать приходится столько, сколько нужно последнему участнику. " +
		"Если кто-то отказался, вещи остальных автоматически возвращаются в поиск, а состав, " +
		"который не сложился, повторно не предлагается.",
	// Про деньги сервис умеет сказать ровно одно, зато точно: платежей в нём нет вообще —
	// ни цен, ни комиссии, ни расчётов между участниками. Про стоимость доставки текст
	// молчит намеренно: в коде на этот счёт нет ничего, и выдумывать ответ нельзя.
	topicPayment: "Сам обмен бесплатный: сервис не берёт комиссию и платежей не проводит вообще. " +
		"Если участники договорились о доплате за разницу в стоимости вещей, они решают это " +
		"между собой в чате обмена — сервис в расчётах не участвует и денег не принимает. " +
		"Именно поэтому просьба перевести деньги на карту — повод показать сообщение модератору " +
		"кнопкой «Пожаловаться», а не переводить.",
	topicAccount: "Вход в сервис — по никнейму и паролю; электронная почта к аккаунту не привязана, " +
		"поэтому автоматического восстановления пароля нет. " +
		"Никнейм, описание и фотографию можно поменять в профиле. " +
		"Если доступ к аккаунту потерян, опишите ситуацию здесь — дальше подключится модератор.",
	topicComplaint: "Пожаловаться на участника можно прямо в чате обмена: у каждого сообщения есть кнопка " +
		"«Пожаловаться», жалоба уходит в очередь модерации. " +
		"Если участник мешает и в других обменах, его можно заблокировать в его профиле — " +
		"тогда сервис не будет собирать с ним общие цепочки. " +
		"Деньги на карту в обмене никто просить не должен: такие сообщения стоит показать модератору.",
	topicOther: "Я не умею отвечать на такое :( \n Уже позвал модератора.",
}

// Generator — то, что умеет спросить модель. Интерфейс нужен тесту: он подменяет
// llm.Client и проверяет разбор ответа без Ollama.
type Generator interface {
	Generate(ctx context.Context, system string, user string, format json.RawMessage) (string, error)
}

type BotRepository interface {
	CreateBotMessage(ctx context.Context, threadID uuid.UUID, botNickname string, body string) (supportmodel.Message, error)
	Escalate(ctx context.Context, threadID uuid.UUID) error
}

type botLogger interface {
	Printf(string, ...any)
}

// Bot отвечает на новое обращение заранее написанным текстом, выбранным моделью.
//
// Вызовы модели живут в очереди с одним потребителем, а не в HTTP-запросе: у
// прод-сервера одно ядро, и инференс занимает то же ядро, на котором API отвечает.
// Очередь заодно сериализует вызовы — двух одновременных инференсов эта машина не
// выдержит.
//
// Сбой модели не ломает пользовательское действие: обращение уже создано и лежит в
// очереди модерации, а отсутствие автоответа — просто отсутствие автоответа.
type Bot struct {
	repository BotRepository
	model      Generator
	logger     botLogger
	jobs       chan botJob
	closeOnce  sync.Once
}

type botJob struct {
	threadID uuid.UUID
	text     string
}

func NewBot(repository BotRepository, model Generator) *Bot {
	return newBot(repository, model, log.Default())
}

func newBot(repository BotRepository, model Generator, logger botLogger) *Bot {
	return &Bot{
		repository: repository,
		model:      model,
		logger:     logger,
		jobs:       make(chan botJob, botQueueCapacity),
	}
}

// Enqueue не блокирует HTTP-запрос и ничего не возвращает: создание обращения уже
// удалось, и провал автоответа его отменять не должен. Нулевой Bot — это выключенная
// фича (пустой OLLAMA_URL), поэтому вызов на нём безопасен.
func (b *Bot) Enqueue(threadID uuid.UUID, subject string, body string) {
	if b == nil {
		return
	}

	select {
	case b.jobs <- botJob{threadID: threadID, text: botInput(subject, body)}:
	default:
		// Буфер исчерпан или очередь закрыта на остановке сервиса.
		b.logger.Printf("support bot queue is full, thread %s gets no automatic reply", threadID)
	}
}

// Run обрабатывает обращения по одному. Закрытая очередь дочитывается до конца,
// отменённый context прерывает воркер немедленно.
func (b *Bot) Run(ctx context.Context) {
	if b == nil {
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case job, open := <-b.jobs:
			if !open {
				return
			}

			if err := b.reply(ctx, job); err != nil {
				b.logger.Printf("support bot failed to answer thread %s: %v", job.threadID, err)
			}
		}
	}
}

// Close прекращает приём задач и даёт воркеру дочитать буфер. Повторный вызов
// безопасен, вызов на нулевом Bot — тоже.
func (b *Bot) Close() {
	if b == nil {
		return
	}

	b.closeOnce.Do(func() { close(b.jobs) })
}

func (b *Bot) reply(ctx context.Context, job botJob) error {
	callCtx, cancel := context.WithTimeout(ctx, botCallTimeout)
	defer cancel()

	topic, err := b.classify(callCtx, job.text)
	if err != nil {
		// Модель не ответила — пользователь не получил вообще ничего, значит человек нужен
		// сразу, а не после того, как пользователь напишет второй раз в пустоту.
		b.escalate(ctx, job.threadID)
		return err
	}
	if topic == topicOther {
		// Бот честно не знает ответа и сам об этом пишет. Ждать повторного сообщения,
		// чтобы позвать человека, здесь незачем — ответа по существу не было.
		b.escalate(ctx, job.threadID)
	}

	_, err = b.repository.CreateBotMessage(ctx, job.threadID, BotNickname, botDisclaimer+botAnswers[topic])
	if errors.Is(err, ErrConflict) {
		// Ноль строк. Штатных причин три: обращение успели закрыть, взять в работу или бот
		// в нём уже отвечал. Нештатная одна и она же самая коварная — служебного
		// пользователя с ником BotNickname нет в базе, и тогда так молчит каждое
		// обращение. Одним запросом эти случаи не различить, поэтому ник сверяется на
		// старте, а здесь он назван в сообщении, чтобы лог не отправлял искать не там.
		b.logger.Printf(
			"support bot skipped thread %s: it no longer waits for an answer, or user %q is missing",
			job.threadID, BotNickname,
		)
		return nil
	}
	if err != nil {
		return fmt.Errorf("save automatic reply: %w", err)
	}

	return nil
}

// escalate помечает обращение как требующее человека. Провал самой пометки не должен
// ломать ответ пользователю, поэтому ошибка только пишется в лог: хуже всего здесь — не
// поставить метку, а не отдать заготовку из-за неё.
func (b *Bot) escalate(ctx context.Context, threadID uuid.UUID) {
	if err := b.repository.Escalate(ctx, threadID); err != nil {
		b.logger.Printf("support bot failed to escalate thread %s: %v", threadID, err)
	}
}

func (b *Bot) classify(ctx context.Context, text string) (string, error) {
	raw, err := b.model.Generate(ctx, botPrompt, text, botSchema)
	if err != nil {
		return "", fmt.Errorf("classify support request: %w", err)
	}

	var answer struct {
		Topic string `json:"topic"`
	}
	if err := json.Unmarshal([]byte(raw), &answer); err != nil {
		return "", fmt.Errorf("decode topic from %q: %w", raw, err)
	}

	if _, known := botAnswers[answer.Topic]; !known {
		// Схема обязывает Ollama вернуть одно из пяти значений, но чужой ответ на этом
		// шве возможен после обновления сервера или модели. Общий текст безопаснее
		// молчания: он никого не вводит в заблуждение.
		return topicOther, nil
	}

	return answer.Topic, nil
}

// botInput склеивает тему с телом обращения и обрезает по границе рун: тема часто
// информативнее первой фразы, а остальное — префилл, за который платит единственное
// ядро.
func botInput(subject string, body string) string {
	text := strings.TrimSpace(subject) + "\n" + strings.TrimSpace(body)
	runes := []rune(text)
	if len(runes) <= botInputLimit {
		return text
	}

	return string(runes[:botInputLimit])
}
