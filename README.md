# Конспекты

Личные конспекты по курсам: **System Design**, **«Глубокий GO»**, **PostgreSQL** (Postgres Pro QPT-16 и DEV1-16) и персональный **ментор** для подготовки к собеседованиям.

Заметки написаны в Obsidian. Прямо на GitHub текст читается нормально, но ссылки-вики `[[...]]` между заметками кликабельны только в Obsidian (или в любом редакторе, который их понимает).

## Содержание

| Курс | Оглавление | Тем |
|---|---|---|
| Глубокий GO | [glubokiy-go/index.md](glubokiy-go/index.md) | 15 |
| [SSL.BAND] [Влад Тен] Систем Дизайн (2025) | [SSL.BAND Систем Дизайн/index.md](SSL.BAND%20Систем%20Дизайн/index.md) | 17 |
| [SuperSliv.biz] System Design [Balun.Courses] | [SuperSliv System Design/index.md](SuperSliv%20System%20Design/index.md) | 8 |
| System Design (merged) | [notes.md](System%20Design%20%28merged%29/notes.md) | 1 |
| 🎯 System Design Interview Simulator | [README.md](System%20Design%20Interview%20Simulator/README.md) | 10 |
| PostgreSQL: Query Performance Tuning (QPT-16) | [postgres-qpt/index.md](postgres-qpt/index.md) | 17 |
| PostgreSQL: Разработчик, часть 1 (DEV1-16) | [postgres-dev1/index.md](postgres-dev1/index.md) | 21 |

## Структура

```
<курс>/
  index.md              # оглавление курса: темы, длительность видео, статус
  <номер - тема>/
    notes.md            # конспект лекции
    frames/             # кадры/слайды из видео
```

## Тренажёр интервью (🎯)

Каталог **`System Design Interview Simulator/`** — тренажёр собеседований по системному дизайну:
10 сценариев с брифом кандидата, вопросами интервьюера, эталонными решениями (скрыты до попытки)
и рубрикой оценки из 6 измерений. Запуск: `cd "System Design Interview Simulator" && go build -o bin/sdsim ./cmd/sdsim && ./bin/sdsim list`.

## Как читать

- **На GitHub** — открой `index.md` нужного курса и переходи по ссылкам в таблице. Текст, таблицы и код читаются как есть; картинки-слайды подтягиваются из `frames/`.
- **В Obsidian** — склонируй репозиторий и открой папку как vault. Тогда работают вики-ссылки между заметками, граф и теги (`course/system-design` и т.п.).

## Ментор для обучения и собеседований

В каталоге [`mentor/`](mentor/) — интерактивный CLI-ментор (Go), который учит и
тренирует по конспектам: направления **Go, PostgreSQL, алгоритмы, System Design,
микросервисы**, режимы — обучение темы, active recall, mock interview, разбор
ошибки, повторение слабых тем, проверка решения, production case. Прогресс
сохраняется в Markdown/JSON. Подробности и запуск — [`mentor/README.md`](mentor/README.md).

## Про источники

Это конспекты — личные текстовые выжимки по лекциям, а не сами курсы. Материалы курсов (видео, слайды целиком) здесь не публикуются.
