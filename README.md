# CoolBlog API 🚀

Мощный и масштабируемый Backend для блог-платформы, написанный на **Go**.

Проект демонстрирует реализацию **Enterprise-паттернов** разработки: **Clean Architecture**, **Transactional Outbox** и **Event-Driven Architecture**. Система спроектирована с упором на надежность данных (Data Consistency) и высокую нагрузку.

🔗 **Repository:** [github.com/TheLionSin/CoolBlog](https://github.com/TheLionSin/CoolBlog)

---

## 🏗 Architecture & Features (Архитектура)

Проект решает проблему надежной доставки событий в микросервисной среде:

* **Clean Architecture**: Строгое разделение на слои `Controllers` -> `Services` -> `Repositories`. Бизнес-логика изолирована от базы данных и HTTP.
* **Transactional Outbox Pattern**: Гарантирует, что события (например, "Пост создан") не потеряются, даже если брокер сообщений недоступен. События сохраняются в БД в одной транзакции с бизнес-данными.
* **Event-Driven (Kafka KRaft)**: Асинхронная коммуникация. Используется современный режим KRaft (без Zookeeper).
* **Reliability & Audit**: Отдельный Consumer-сервис вычитывает события и формирует Audit Log. Поддерживается **Graceful Shutdown** и обработка "отставания" (Lag) консьюмера.
* **Caching**: Использование **Redis** для кэширования "горячих" данных (списки постов, сессии).
* **Security**: JWT авторизация (Access + Refresh tokens) с ротацией токенов в PostgreSQL.

---

## 🛠 Tech Stack (Стек)

* **Language**: Go 1.25.3
* **Web Framework**: Gin
* **Database**: PostgreSQL (GORM), Redis
* **Message Broker**: Apache Kafka (KRaft mode)
* **Infrastructure**: Docker, Docker Compose
* **Tooling**: Makefile, Golangci-lint

---

## 🚀 Getting Started (Запуск)

Для удобства все команды автоматизированы через `Makefile`.

### Prerequisites
* Docker & Docker Compose
* Make (optional, commands can be run manually)

### Installation

1.  **Clone the repository:**
    ```bash
    git clone [https://github.com/TheLionSin/CoolBlog.git](https://github.com/TheLionSin/CoolBlog.git)
    cd CoolBlog
    ```

2.  **Setup Environment:**
    Убедитесь, что файл `.env.docker` создан (на основе примера).

3.  **Build & Run:**
    Эта команда соберет Docker-образы и запустит всю инфраструктуру (Kafka, Postgres, Redis, API, Workers).
    ```bash
    make up
    ```
    > *Wait a few moments for Kafka to initialize topics automatically.*

4.  **View Logs:**
    ```bash
    make logs
    ```

5.  **Stop:**
    ```bash
    make down
    ```

---

## 🔌 API Endpoints

Основные маршруты API:

### Auth (Авторизация)
* `POST /auth/register` — Регистрация (Creates `UserRegistered` event)
* `POST /auth/login` — Вход (Returns Access + Refresh tokens)
* `POST /auth/refresh` — Обновление токенов
* `POST /auth/logout` — Выход

### Users (Пользователи)
* `GET /users/me` — Профиль текущего пользователя
* `PUT /users/me` — Обновление профиля

### Posts (Посты)
* `GET /posts` — Список постов (Cached)
* `GET /posts/:slug` — Получить пост детально
* `POST /posts` — Создать пост (Auth required)
* `PUT /posts/:slug` — Обновить пост
* `DELETE /posts/:slug` — Удалить пост

### Interactions (Лайки и Комментарии)
* `GET  /posts/:slug/comments` — Список комментариев
* `POST /posts/:slug/comments` — Добавить комментарий
* `DELETE /posts/comments/:id` — Удалить комментарий
* `GET  /posts/:slug/likes` — Количество лайков
* `POST /posts/:slug/like` — Поставить лайк
* `DELETE /posts/:slug/like` — Убрать лайк

---

## 📂 Project Structure

```text
.
├── cmd
│   ├── api          # Main API Server
│   ├── consumer     # Kafka Consumer (Audit Log)
│   └── publisher    # Outbox Publisher (Postgres -> Kafka)
├── config           # Environment & DB config
├── internal
│   ├── adapters     # HTTP Controllers (Gin handlers)
│   ├── ports        # Interfaces: Services & Repositories
│   ├── models       # Database structs (GORM)
│   ├── dto          # Request/Response structs
│   ├── events       # Kafka Event definitions
│   ├── middleware   # Auth & Rate Limiting
│   ├── stores       # Redis implementations
│   └── validators   # Custom validation logic
└── docker-compose.*.yml