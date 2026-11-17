---
type: scenario
slug: feature-flag-platform
order: 10
level: senior
focus: "read-heavy, правила таргетинга, low-latency SDK, staleness"
tags: [course/system-design, interview/scenario]
created: 2026-11-17
solution: "../solutions/10-feature-flag-platform.md"
---

# Сценарий 10 — Платформа фиче-флагов

> [!abstract] Бриф
> Спроектируйте платформу фиче-флагов: команды управляют флагами и правилами таргетинга,
> а сервисы через SDK быстро проверяют значение флага для конкретного пользователя.

## Что дано (функциональные требования)

- CRUD флагов, окружения (dev/stage/prod).
- Правила таргетинга: процентное раскатывание, по атрибутам, allow/deny списки.
- Evaluation API/SDK для сервисов и клиентов.
- История изменений и аудит, возможность мгновенного kill-switch.

## Нефункциональные требования (даны частично)

- Read-heavy: проверок флага в разы больше, чем изменений конфигурации.
- Evaluation — сверхбыстрая (микросекунды локально, миллисекунды по сети).
- Низкая задержка применения изменений (staleness), высокая доступность.

> [!question] Вопросы интервьюера на старте
> 1. Сколько проверок флага в секунду?
> 2. Какая допустимая задержка применения изменения флага?
> 3. Что важнее при недоступности конфига — fail-open (старое значение) или fail-closed?

## Ваша попытка — заполните разделы

Запишите ответ письменно (например, в `attempts/10-feature-flag-platform.md`). Раскройте:

1. **Уточняющие вопросы** — QPS проверок, staleness, fail-open/closed, окружения.
2. **Оценка нагрузки** — evaluation QPS, размер конфигурации, объём аудита.
3. **API** — evaluation (bulk), CRUD rules, stream/config delivery.
4. **Модель данных** — flag, rule, segment, environment, audit.
5. **Архитектура** — control plane (управление) vs data plane (evaluation/SDK).
6. **Масштабирование** — кэш/SDK локально, CDN, поток конфигов, шардинг.
7. **Отказоустойчивость** — недоступность конфига, деградация SDK, консистентность кэша.
8. **Consistency** — staleness, версии конфига, атомарность правил.
9. **Bottlenecks** — массовая рассылка конфигов, hot flags, evaluation на клиенте.
10. **Trade-offs** — pull vs push конфигов, SDK-local vs server-side eval, fail-open vs fail-closed.

> [!danger] Стоп-гейт
> Не открывайте `solutions/10-feature-flag-platform.md`, пока не зафиксировали свой ответ.
