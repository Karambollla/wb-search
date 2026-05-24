Сервис для виджета Wildberries "Сейчас ищут". Приложение читает поисковые события из NATS, считает популярные запросы за последние 5 минут и отдает топ через HTTP и gRPC

## Архитектура

Архитектура гексагональная (ports & adapters):

- `internal/core`: доменные модели, порты, нормализация, стоп-лист, антифрод, sliding window и top cache
- `internal/adapters/nats`: адаптер входящего потока событий из брокера
- `internal/adapters/http`: HTTP API
- `internal/adapters/grpc`: gRPC
- `internal/adapters/metrics`: Prometheus метрики
- `cmd/server`: сборка зависимостей, запуск серверов и graceful shutdown

БД в hot path не используется. Виджет real-time, окно всего 5 минут, а чтений ожидается в 10-50 раз больше, чем входящих событий. Поэтому агрегаты хранятся in-memory, а топ возвращается моментально из кеша

## Локальный запуск

```bash
make up
```

После запуска:

- HTTP: `localhost:8080`
- gRPC: `localhost:9090`
- NATS: `localhost:4222`

Отправить несколько тестовых событий в NATS:

```bash
nats pub search.events '{"query":"кроссовки nike","user_id":"u1","session_id":"s1"}'
nats pub search.events '{"query":"кроссовки nike","user_id":"u2","session_id":"s2"}'
nats pub search.events '{"query":"iphone 15","user_id":"u3","session_id":"s3"}'
```

Получить топ через 1-2 секунды после отправки:

```bash
curl 'http://localhost:8080/v1/top?limit=10'
```

Проверить gRPC:

```bash
grpcurl -plaintext -d '{"limit":10}' localhost:9090 searchv1.TopService/GetTop
grpcurl -plaintext -d '{"term":"spam"}' localhost:9090 searchv1.StopListService/AddStopWord
```

Коллекция для Postman лежит в `postman/wb-search.postman_collection.json`

## API

HTTP:

- `GET /v1/top?limit=10`
- `GET /v1/stoplist`
- `POST /v1/stoplist` с телом `{"term":"spam"}`
- `DELETE /v1/stoplist/{term}`
- `GET /healthz`
- `GET /metrics`

gRPC-контракт описан в `api/proto/searchv1/search.proto`.

## Примеры запросов

Healthcheck:

```bash
curl http://localhost:8080/healthz
```

Отправить события в брокер:

```bash
nats pub search.events '{"query":"кроссовки nike","user_id":"u1","session_id":"s1"}'
nats pub search.events '{"query":"кроссовки nike","user_id":"u2","session_id":"s2"}'
nats pub search.events '{"query":"iphone 15","user_id":"u3","session_id":"s3"}'
```

Получить топ запросов:

```bash
curl 'http://localhost:8080/v1/top?limit=10'
```

Посмотреть стоп-лист:

```bash
curl http://localhost:8080/v1/stoplist
```

Добавить слово в стоп-лист:

```bash
curl -X POST http://localhost:8080/v1/stoplist \
  -H 'Content-Type: application/json' \
  -d '{"term":"iphone 15"}'
```

Удалить слово из стоп-листа:

```bash
curl -X DELETE http://localhost:8080/v1/stoplist/iphone%2015
```

Получить метрики:

```bash
curl http://localhost:8080/metrics
```

gRPC top:

```bash
grpcurl -plaintext -d '{"limit":10}' localhost:9090 searchv1.TopService/GetTop
```

gRPC стоп-лист:

```bash
grpcurl -plaintext localhost:9090 searchv1.StopListService/ListStopWords
grpcurl -plaintext -d '{"term":"spam"}' localhost:9090 searchv1.StopListService/AddStopWord
grpcurl -plaintext -d '{"term":"spam"}' localhost:9090 searchv1.StopListService/DeleteStopWord
```

## Контракт брокера

NATS subject: `search.events`.

Payload:

```json
{
  "query": "Кроссовки",
  "timestamp": "2026-05-23T12:00:00Z",
  "user_id": "u123",
  "session_id": "s456",
  "ip": "203.0.113.10",
  "source": "web"
}
```

`query` обязателен. `timestamp` нужен для попадания события в 5-минутное окно; если поле не передано, используется серверное время. `user_id`, `session_id` и `ip` нужны для базового антифрода: по одному тексту запроса невозможно отличить реальный спрос от накрутки ботом

## Бизнес-логика

- Запросы нормализуются: trim, lowercase, схлопывание пробелов, ограничение максимальной длины
- Sliding window реализован как 60 бакетов по 5 секунд
- Старые бакеты удаляются через перезапись слот в буфере
- Фоновый воркер раз в 1-2 секунды пересобирает `Top-100`
- HTTP и gRPC read path только отрезают готовый отсортированный cache по `limit`
- Стоп-лист динамический и copy-on-write, поэтому чтения не блокируются на каждом событии
- Антифрод упрощенный, `query + user_id/session_id/ip` учитывается не чаще одного раза за короткий TTL
- Если у события есть `user_id`, разные `session_id` одного пользователя не увеличивают счетчик повторно

## Trade-offs

- Состояние теряется при рестарте. Для realtime виджета это ок, но в проде можно добавить Redis
- Топ eventually consistent в пределах интервала, зато не пересчитывается на каждый read request
- Граница 5 минутного окна приблизительная в пределах размера бакета
- Антифрод по сути небольшой фильтр, а не полноценная система

## Тесты и бенчмарки

Юнит тесты и бенчмарки:

```bash
make unit
make bench
```

E2E тесты в Docker Compose:

```bash
make test
```

`make test` поднимает контейнеры `app` и `nats`, запускает отдельный контейнер `tests`, отправляет события в NATS и проверяет HTTP API, стоп лист и метрики

Go бенчмарки проверяют hot path внутри core:

```text
BenchmarkServiceIngest      ~3006 ns/op
BenchmarkServiceGetTop      ~13.01 ns/op
```

Для внешнего тестирования с нагрузкой HTTP API можно использовать `hey`:

```bash
hey -z 30s -c 100 'http://localhost:8080/v1/top?limit=10'
```

Метрики Prometheus доступны по адресу:

```bash
curl http://localhost:8080/metrics
```
