# Task Manager

REST API для управления задачами внутри команд.

## Стек

- Go
- MySQL 8
- Redis
- Docker / Docker Compose
- JWT
- OpenAPI 3.0

## Возможности

- регистрация и авторизация пользователей;
- JWT-аутентификация;
- создание команд;
- ролевая модель `owner / admin / member`;
- приглашение пользователей и изменение ролей;
- создание, назначение и редактирование задач;
- фильтрация и пагинация списка задач;
- optimistic locking для защиты от потерянных обновлений;
- история изменений задач;
- комментарии;
- кеширование списков задач в Redis;
- аналитика по команде;
- SQL-миграции;
- интеграционный тест SQL-отчёта.

## Структура проекта

```text
cmd/app/              # точка входа
internal/
  config/             # конфигурация
  handler/            # HTTP handlers
  middleware/         # middleware
  model/              # модели
  repository/         # работа с MySQL и Redis
  service/            # бизнес-логика
migrations/           # SQL-миграции
docs/                 # OpenAPI
```

## Конфигурация

Создайте `.env` на основе `.env.example`.

Основные переменные окружения:

```env
MYSQL_ROOT_PASSWORD=...
MYSQL_DATABASE=task_manager
MYSQL_USER=task_user
MYSQL_PASSWORD=...

JWT_SECRET=...
HTTP_PORT=8080
```

Секреты и пароли не хранятся в репозитории.

## Запуск через Docker Compose

```bash
docker compose up --build
```

После запуска API доступен по адресу:

```text
http://localhost:8080
```

Docker Compose запускает:

- приложение;
- MySQL;
- Redis;
- миграции базы данных.

Для запуска в фоне:

```bash
docker compose up -d --build
```

Остановка:

```bash
docker compose down
```

## Миграции

Миграции находятся в директории:

```text
migrations/
```

При запуске через Docker Compose миграции автоматически применяются сервисом `migrate` до старта приложения.

Для ручного применения:

```bash
migrate \
  -path ./migrations \
  -database "mysql://task_user:task_password@tcp(127.0.0.1:3306)/task_manager" \
  up
```

## Авторизация

Защищённые endpoint'ы требуют JWT:

```http
Authorization: Bearer <token>
```

### Регистрация

```http
POST /api/v1/register
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "password123",
  "name": "User"
}
```

### Авторизация

```http
POST /api/v1/login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "password123"
}
```

## Примеры запросов

### Создание команды

```http
POST /api/v1/teams
Authorization: Bearer <token>
Content-Type: application/json

{
  "name": "Backend Team"
}
```

### Приглашение участника

```http
POST /api/v1/teams/1/invite
Authorization: Bearer <token>
Content-Type: application/json

{
  "user_id": 2,
  "role": "member"
}
```

### Создание задачи

```http
POST /api/v1/tasks
Authorization: Bearer <token>
Content-Type: application/json

{
  "team_id": 1,
  "title": "Implement API",
  "description": "Implement task endpoints",
  "status": "todo",
  "assignee_id": 2
}
```

### Получение задач с фильтрацией

```http
GET /api/v1/tasks?team_id=1&status=todo&assignee_id=2&limit=20&offset=0
Authorization: Bearer <token>
```

### Обновление задачи

При обновлении используется поле `version` для optimistic locking.

```http
PUT /api/v1/tasks/1
Authorization: Bearer <token>
Content-Type: application/json

{
  "status": "done",
  "version": 1
}
```

Если задача уже была изменена и передана устаревшая версия, API возвращает:

```text
409 Conflict
```

### Аналитика команды

```http
GET /api/v1/teams/1/stats
Authorization: Bearer <token>
```

Доступна только пользователям с ролями `owner` и `admin`.

## Кеширование

Список задач кешируется в Redis на 5 минут.

Ключ кеша учитывает:

- команду;
- статус;
- исполнителя;
- `limit`;
- `offset`.

После создания или изменения задачи кеш соответствующей команды инвалидируется.

## Закрытие задач

Статус `done` считается закрытым состоянием задачи.

При переходе задачи в `done` заполняется `closed_at`. При переводе из `done` в другой статус `closed_at` очищается.

## Тесты

Обычный запуск тестов:

```bash
go test ./...
```

Интеграционный тест SQL-отчёта:

```bash
go test -tags=integration ./internal/repository/... -v
```

Для интеграционного теста необходимо задать:

```env
TEST_MYSQL_DSN=task_user:task_password@tcp(127.0.0.1:3306)/task_manager?parseTime=true
```

## OpenAPI

Спецификация API находится в:

```text
docs/openapi.yaml
```