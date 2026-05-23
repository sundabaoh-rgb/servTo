# Tanki

Это проект клона игры Tanki Online, реализованный на языке **Go** с использованием микросервисного подхода и контейнеризации **Docker**.

## Технологический стек
* **Backend:** Go (1.24)
* **Database:** PostgreSQL 16
* **Frontend:** HTML/CSS/JS
* **Communication:** WebSockets (Gorilla)
* **Routing:** go-chi/chi
* **Infrastructure:** Docker & Docker Compose, FRP (для туннелирования)

## Функционал
* Регистрация и аутентификация пользователей.
* Система лобби для поиска матчей.
* Реальное время: игровой процесс через WebSockets.
* Сохранение состояния игры в PostgreSQL.

---

## Интерфейс

### Страница регистрации
*(Здесь будет скриншот формы регистрации)*
### Лобби
*(Здесь будет скриншот списка комнат и выбора матча)*
### Игровой процесс
*(Здесь будет скриншот геймплея)*
---

## Запуск проекта

Для развертывания проекта локально убедитесь, что у вас установлены [Docker](https://www.docker.com/) и [Docker Compose](https://docs.docker.com/).

1. **Клонируйте репозиторий:**
   ```bash
   git clone [https://github.com/sundabaoh-rgb/tankionline.git](https://github.com/sundabaoh-rgb/tankionline.git)
   cd tankionline
2. **Запустите контейнер:**
   ```bash
   docker compose up -d --build
3. **Откройте в браузере:**
   [Tanki](http://localhost:8080/)
