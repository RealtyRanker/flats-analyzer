# flats-analyzer

Сервис-анализатор объявлений об аренде. Читает сообщения о новых квартирах из Kafka, сопоставляет каждое объявление с активными подписками пользователей в PostgreSQL и отправляет подходящие квартиры через `users-notifier`. Уже отправленные объявления не дублируются.

## Место в архитектуре

```
realty-parser  →  Kafka (realty.flats)  →  flats-analyzer  →  users-notifier  →  Telegram
                                               ↕
                                          PostgreSQL
                              (user_subscriptions, user_sent_messages)
```

## Что делает сервис

1. Потребляет сообщения из Kafka-топика `realty.flats` (consumer group `flats-analyzer`)
2. Для каждой квартиры загружает все активные подписки из таблицы `user_subscriptions`
3. Проверяет, подходит ли квартира под фильтры подписки: цена, площадь, комнатность, минимальный score
4. Проверяет по таблице `user_sent_messages`, не было ли это объявление уже отправлено данному пользователю
5. Если нет — отправляет сообщение через HTTP-запрос к `users-notifier /send` и записывает факт отправки
6. Экспортирует Prometheus-метрики

Сервис идемпотентен: при рестарте consumer group продолжает читать с сохранённого offset, а `user_sent_messages` гарантирует отсутствие дублей.

## Конфигурация

```yaml
kafka:
  brokers:
    - "realty-kafka:9092"
  topic: "realty.flats"
  group_id: "flats-analyzer"   # offset хранится в Kafka по этому ID

database:
  dsn: "postgres://realty_parser:password@realty-postgres:5432/realty_parser?sslmode=disable"

notifier:
  base_url: "http://users-notifier:8080"   # адрес users-notifier внутри Docker-сети

logging:
  level: "info"    # debug / info / warn / error
  file_path: "/var/log/flats-analyzer/app.log"

metrics:
  port: 9090
```

## Фильтрация подписок

Квартира подходит под подписку, если выполнены все заданные условия:

| Поле подписки | Условие | Пропускается если |
|---|---|---|
| `min_price` | `flat.price >= min_price` | `min_price = 0` |
| `max_price` | `flat.price <= max_price` | `max_price = 0` |
| `min_area` | `flat.total_area >= min_area` | `min_area = 0` |
| `max_area` | `flat.total_area <= max_area` | `max_area = 0` |
| `rooms` | `flat.room_number ∈ rooms` | массив пустой |
| `min_score` | `flat.flat_score >= min_score` | `min_score = 0` |

Подписки с `is_active = FALSE` не обрабатываются.

## Метрики

Доступны на порту `9093` (на хосте):

```bash
curl http://localhost:9093/metrics
curl http://localhost:9093/healthz
```

| Метрика | Тип | Описание |
|---|---|---|
| `analyzer_flats_consumed_total` | Counter | Квартир прочитано из Kafka |
| `analyzer_subscriptions_matched_total` | Counter | Совпадений квартира × подписка |
| `analyzer_messages_sent_total` | Counter | Успешно отправленных уведомлений |
| `analyzer_messages_failed_total` | Counter | Ошибок при отправке |
| `analyzer_flat_process_duration_seconds` | Histogram | Время обработки одного сообщения |

---

## Запуск в Docker

Сервис запускается в сети `realty-net`. Перед запуском должны быть подняты PostgreSQL, Kafka и `users-notifier`.

### Порядок запуска

```bash
# 1. PostgreSQL
cd /путь/к/realty-parser && bash psql_setup.sh

# 2. Kafka
cd /путь/к/realty-parser && bash kafka_setup.sh

# 3. users-notifier
cd /путь/к/users-notifier && bash server_setup.sh

# 4. flats-analyzer
cd /путь/к/flats-analyzer && bash server_setup.sh
```

### Управление контейнером

```bash
# Логи
docker logs -f flats-analyzer

# Остановить
docker stop flats-analyzer
```

### Добавление тестовой подписки

```sql
-- docker exec -it realty-postgres psql -U realty_parser -d realty_parser

INSERT INTO user_subscriptions (chat_id, min_price, max_price, min_area, rooms, min_score)
VALUES (123456789, 50000, 90000, 38, '{2,3}', 0);
```

После этого все новые квартиры из Kafka, подходящие под фильтр, будут отправлены в чат `123456789`.
