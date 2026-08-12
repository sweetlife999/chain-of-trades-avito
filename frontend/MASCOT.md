# Маскот «У»

Интерактивный маскот встроен без дополнительных runtime-зависимостей: визуал — SVG + CSS-анимации, состояние — Redux Toolkit.

## Где используется

- Лендинг: приветствие при открытии, крупный hero-вариант.
- Чат цепочки обмена: быстрые подсказки, реакция на набор текста, отправку/получение сообщений и системные события обмена.
- Поддержка: компактный помощник и быстрые фразы для обращения.

## Состояния

`idle`, `hello`, `listening`, `thinking`, `hint`, `happy`, `celebrate`, `concerned`.

Режимы показа: `ambient`, `attention`, `dialog`.

## Основные файлы

- `src/Components/UI/Mascot/Mascot.tsx` — SVG-персонаж.
- `src/Components/UI/Mascot/Styles.module.scss` — анимации и адаптив.
- `src/Features/Mascot/mascot.types.ts` — типы состояний/событий.
- `src/Features/Mascot/mascotEvents.ts` — карта событие → реакция.
- `src/Features/Mascot/mascotSlice.ts` — Redux-состояние.
- `src/Hooks/useMascot.ts` — публичный API для компонентов.

## Как вызвать реакцию

```ts
const { reactTo } = useMascot();
reactTo("HINT_SELECTED");
```

Чтобы изменить поведение продукта, достаточно поменять реакцию в `mascotEvents.ts`; страницы не должны напрямую выбирать CSS-анимацию.

## Проверка

```bash
npm ci
npm run check
npm run dev
```

`node_modules` намеренно не включён в итоговый архив: исходный архив содержал Windows-сборку нативной зависимости Rolldown, поэтому зависимости следует установить на целевой машине через lock-файл.
