# Погружение в архитектуру Forgejo и Образовательного расширения

Этот документ предназначен для разработчика, желающего понять внутреннее устройство проекта "с нуля". Он охватывает архитектуру платформы Forgejo и детали реализации образовательного модуля (Edu Extension).

---

## Часть 1: Архитектура Forgejo

Forgejo (форк Gitea) — это монолитное приложение на Go, использующее стандартную трехуровневую архитектуру (MVC).

### 1.1 Структура папок

*   **`cmd/`**: Точки входа (main). Команда `web` запускает сервер.
*   **`conf/`**: Конфигурация (шаблоны `app.ini`).
*   **`routers/`**: **Контроллеры (C)**. Здесь живут HTTP-хендлеры.
    *   `routers/web/`: Обработчики HTML-страниц (UI).
    *   `routers/api/`: Обработчики REST API (v1).
    *   `routers/private/`: Внутренний API для взаимодействия с git-хуками и раннерами.
*   **`models/`**: **Данные (M)**. Структуры БД (Xorm beans) и базовые запросы.
    *   `models/db/`: Базовый движок и контекст БД.
    *   `models/user/`, `models/repo/`, `models/issue/`: Доменные модели.
*   **`services/`**: **Бизнес-логика**. "Толстые" сервисы, выполняющие сложные операции (создание репозитория с инициализацией git, отправка уведомлений и т.д.).
    *   *Правило*: Роутеры вызывают Сервисы, Сервисы вызывают Модели.
*   **`modules/`**: Утилиты и библиотеки (Middleware, Git-обертки, Логгер, Шаблонизатор).
    *   `modules/git/`: Взаимодействие с git-командой.
    *   `modules/repository/`: Утилиты для работы с репозиториями, включая `InternalPushingEnvironment`.
*   **`templates/`**: **Представление (V)**. Go-шаблоны (`.tmpl`), использующие синтаксис `{{...}}`.

### 1.2 Жизненный цикл запроса (Request Lifecycle)

Когда приходит HTTP-запрос (например, `GET /user/repo`):

1.  **Chi Router (`routers/routes.go`)**: Маршрутизатор определяет, какой хендлер вызвать.
2.  **Middleware (`modules/context/`)**:
    *   Оборачивает `http.Request` в `context.Context` (свой, "толстый" контекст Forgejo).
    *   Загружает текущего пользователя (`ctx.Doer`).
    *   Загружает репозиторий, если URL содержит `/user/repo` (`ctx.Repo`).
3.  **Handler (`routers/web/...`)**:
    *   Получает данные из `ctx`.
    *   Выполняет проверки прав.
    *   Вызывает `services` или `models` для получения данных.
    *   Кладет данные в `ctx.Data["Key"] = Value`.
    *   Вызывает рендер: `ctx.HTML(200, "template_name")`.
4.  **Template**: Генерирует HTML, используя данные из `ctx.Data`.

---

## Часть 2: Образовательное расширение (Edu Extension)

Мы внедрили полноценный LMS-модуль ("LMS внутри Git") прямо в монолит Forgejo. Весь код расширения изолирован в отдельных директориях, чтобы минимизировать влияние на ядро.

### 2.1 Где лежит код?

| Слой | Путь | Назначение |
|------|------|------------|
| Модели + Сервисы + DAL | `internal/edu/` | Ядро бизнес-логики, интерфейсы, данные |
| HTTP-хендлеры | `routers/web/edu/` | Привязка к URL, обработка запросов |
| Шаблоны | `templates/edu/` | UI (Fomantic UI) |
| Интеграционные тесты | `tests/integration/edu_assignments_test.go` | SQLite-based Go тесты (15 тестов) |
| E2E тесты | `tests/e2e/edu.test.e2e.ts` | Playwright тесты (8 тестов) |
| Фикстуры | `models/fixtures/user_role.yml` | Тестовые данные для edu ролей |
| Docker Dev | `Dockerfile.dev`, `docker-compose.dev.yml`, `docker/app.ini` | Локальная среда разработки |
| Документация | `edu-docs/` | Руководства для разработчиков и пользователей |

### 2.2 Архитектура модуля

Модуль следует чистой слоёной архитектуре:

```
HTTP Handler (routers/web/edu/)
    │
    ▼
EducationalService (interface в service.go)
    │
    ├──> Repository (interface в service.go, реализация в repository_*.go)
    │        └──> SQLRunner (database/sql) через squirrel
    │
    ├──> RepoForker (interface в service.go)
    │        └──> ForgejoAdapter (adapter.go) → Forgejo core services
    │
    └──> UserCreator (interface в service.go)
             └──> ForgejoAdapter (adapter.go) → user_model.*
```

**Ключевые абстракции:**
- **`EducationalService`** — интерфейс бизнес-логики (~30 методов)
- **`Repository`** — интерфейс DAL (~40 методов для CRUD всех сущностей)
- **`RepoForker`** — абстракция над git-операциями (fork, sync, получение репо)
- **`UserCreator`** — абстракция над управлением пользователями
- **`ForgejoAdapter`** — мост к ядру Forgejo, реализует `RepoForker` и `UserCreator`

Все интерфейсы определены в `internal/edu/service.go`.

### 2.3 Модель данных (10 таблиц)

Таблицы создаются автоматически через Xorm `e.Sync()` при старте приложения в `init.go`. Миграций нет. Большинство моделей определяют метод `TableName()` для задания имени таблицы с префиксом `edu_`.

#### Основные сущности:

```
┌──────────────┐     ┌─────────────────────┐     ┌────────────────┐
│  edu_courses │────<│ edu_course_enrollments│>────│  user (Forgejo) │
│              │     │                     │     │                │
│ id           │     │ course_id           │     │ id             │
│ name         │     │ user_id             │     └────────────────┘
│ description  │     │ role (student/      │
│ creator_id   │     │        teacher/admin)│           ▲
│ org_id       │     └─────────────────────┘           │
│ start_unix   │                                       │
│ end_unix     │     ┌────────────────┐     ┌──────────┴───────┐
└──────┬───────┘     │ edu_assignments│     │ edu_submissions  │
       │             │                │────<│                  │
       └────────────>│ course_id      │     │ assignment_id    │
                     │ repo_id ──────>│     │ user_id          │
                     │ title          │     │ student_repo_id  │
                     │ description    │     │ status           │
                     │ deadline_unix  │     │ grade (-1=нет)   │
                     └────────────────┘     │ comment          │
                                            │ graded_by_id     │
                     ┌────────────────┐     │ graded_unix      │
                     │edu_test_results│<────┤                  │
                     │                │     └──────────────────┘
                     │ submission_id  │
                     │ commit_sha     │
                     │ score (0-100)  │
                     │ details        │
                     └────────────────┘
```

#### Вспомогательные таблицы:

| Таблица | Назначение |
|---------|------------|
| `user_role` | Глобальная роль пользователя (student/teacher/admin). **Внимание:** эта таблица управляется через Xorm ORM напрямую (`role.go`), а не через squirrel, поэтому **не имеет** префикса `edu_`. |
| `edu_import_draft` | Черновик CSV-импорта (храним сырой CSV) |
| `edu_import_draft_row` | Строки черновика импорта (ФИО, email, username, status) |
| `bulk_fork_task` | Задача массового форка (прогресс: completed/failed/total) |
| `sync_fork_task` | Задача массовой синхронизации форков (synced/skipped/failed) |

> **Важно о именовании таблиц:** Модели `Course`, `Assignment`, `Submission`, `TestResult`, `CourseEnrollment`, `ImportDraft`, `ImportDraftRow` определяют метод `TableName()` в `models.go`, чтобы Xorm создавал таблицы с `edu_` префиксом (соответствуя squirrel-запросам). `UserRole` не имеет `TableName()`, поэтому Xorm создаёт таблицу как `user_role`. `BulkForkTask` и `SyncForkTask` также не имеют `TableName()`, и squirrel-запросы к ним используют имена `bulk_fork_task` / `sync_fork_task` (совпадают с дефолтным поведением Xorm).

#### Статусы Submission:
- `started` — студент взял задание, форк создан
- `submitted` — (зарезервирован для ручной сдачи)
- `passed` — CI/CD прошёл успешно
- `failed` — CI/CD упал
- `graded` — преподаватель выставил оценку

### 2.4 Файлы `internal/edu/`

| Файл | Содержимое |
|------|------------|
| `models.go` | Все структуры данных (Course, Assignment, Submission, TestResult, ...) |
| `service.go` | Интерфейсы EducationalService, Repository, RepoForker, UserCreator; конструктор |
| `repository.go` | SQLRunner, dbRepository, CRUD для assignments и submissions |
| `repository_courses.go` | CRUD для courses |
| `repository_enrollments.go` | CRUD для enrollments |
| `repository_submissions.go` | Дополнительные методы submissions (GetByRepoID) |
| `repository_test_results.go` | CRUD для test results |
| `repository_import.go` | CRUD для import drafts и draft rows |
| `repository_bulk_fork.go` | CRUD для bulk fork tasks |
| `repository_sync_fork.go` | CRUD для sync fork tasks |
| `repository_ext.go` | Расширенные запросы (GetAssignmentsForUser через JOIN enrollment) |
| `service_courses.go` | Бизнес-логика курсов |
| `service_join.go` | Логика "взять задание" (fork + create submission) |
| `service_import.go` | Логика CSV импорта (upload → preview → execute) |
| `service_bulk_fork.go` | Логика массового форка |
| `service_sync_fork.go` | Логика массовой синхронизации |
| `service_grading.go` | Логика оценивания |
| `service_ext.go` | Расширенные сервисные методы |
| `csv_import.go` | Парсер CSV (BOM, Windows-1251, авто-разделитель) |
| `translit.go` | Транслитерация кириллицы → латиница (ГОСТ 7.79-2000) |
| `adapter.go` | ForgejoAdapter — мост к ядру Forgejo |
| `notifier.go` | EduNotifier — обработка событий CI/CD |
| `role.go` | Управление глобальными ролями через Xorm ORM (таблица `user_role`) |
| `init.go` | Инициализация: sync схемы, создание репозитория, регистрация нотификатора |
| `mock_test.go` | Моки для unit-тестов |
| `*_test.go` | Unit-тесты для каждого компонента (19 файлов) |

### 2.5 Ключевые сценарии

#### А. Управление курсами

Файлы: `routers/web/edu/courses.go`, `templates/edu/course_list.tmpl`, `course_detail.tmpl`, `course_form.tmpl`

1. Преподаватель заходит на `/edu/teacher/courses` — видит список своих курсов.
2. Создаёт курс через `/edu/teacher/courses/new` (название, описание, даты).
3. На странице курса (`/edu/teacher/courses/{id}`) видит список участников и может:
   - Добавить студента вручную (по username)
   - Импортировать студентов из CSV
   - Удалить участника
4. Курс привязывается к заданиям — задание всегда принадлежит курсу.

#### Б. CSV-импорт студентов

Файлы: `routers/web/edu/import.go`, `internal/edu/csv_import.go`, `internal/edu/translit.go`, `service_import.go`

Трёхшаговый процесс:

1. **Upload**: Преподаватель загружает CSV (поддерживает UTF-8, Windows-1251, BOM, запятая и точка с запятой как разделитель).
2. **Preview**: Система парсит CSV, транслитерирует ФИО в username (`Иванов Иван` → `ivanov-i`), показывает таблицу для редактирования (можно исправить username/email).
3. **Execute**: Система создаёт пользователей (с автогенерацией пароля), записывает их в курс. Показывает таблицу с логинами и паролями для раздачи.

Если пользователь уже существует — просто записывается в курс без создания нового.

#### В. Создание задания

Файлы: `routers/web/edu/assignments.go`, `templates/edu/assignment_new.tmpl`

1. Преподаватель создаёт обычный репозиторий-шаблон в Forgejo.
2. Заходит на `/edu/teacher/assignments/new`, выбирает курс и репозиторий, заполняет название, описание, дедлайн.
3. Запись создаётся в `edu_assignments` с привязкой к `course_id` и `repo_id`.

#### Г. Студент берёт задание (Join)

Файлы: `routers/web/edu/assignments.go` → `JoinAssignment`, `internal/edu/service_join.go`

1. Студент видит список своих заданий на `/edu/student/assignments` (только по курсам, в которых он записан).
2. Нажимает "Start Assignment" на `/edu/student/assignments/{id}`.
3. Сервис вызывает `RepoForker.ForkRepositoryAndUpdates` — создаёт форк шаблона в namespace студента.
4. Создаётся запись `Submission` со статусом `started` и ссылкой на форк.
5. Студент перенаправляется на страницу задания, где видит ссылку на свой репозиторий.

#### Д. Массовый форк (Bulk Fork)

Файлы: `routers/web/edu/bulk_fork.go`, `internal/edu/service_bulk_fork.go`

1. На странице submissions преподаватель нажимает "Fork for All Students".
2. Система берёт всех `student` enrollments курса, для каждого:
   - Проверяет, нет ли уже submission.
   - Создаёт форк шаблона → создаёт submission.
3. Прогресс записывается в `BulkForkTask` (completed/failed/total).
4. Результат показывается на странице submissions (progress bar, error log).

#### Е. Синхронизация форков (Sync Forks)

Файлы: `routers/web/edu/sync_fork.go`, `internal/edu/service_sync_fork.go`, `adapter.go`

1. Преподаватель обновляет шаблон-репозиторий (добавляет тесты, README и т.д.).
2. Нажимает "Sync All Forks" на странице submissions.
3. Для каждого форка студента выполняется `git push` из шаблона в форк.

**Ключевое решение**: используется `InternalPushingEnvironment` из `modules/repository/env.go`, что устанавливает переменную `GITEA_INTERNAL_PUSH=true`. Это приводит к тому, что git-хуки Forgejo пропускают проверку branch protection. Без этого `push` в чужой форк блокировался бы защитой веток.

```go
// adapter.go — ключевой фрагмент
func (a *ForgejoAdapter) SyncFork(ctx context.Context, doer *user_model.User, forkRepo *repo_model.Repository, branch string) error {
    return git.Push(ctx, forkRepo.BaseRepo.RepoPath(), git.PushOptions{
        Remote: forkRepo.RepoPath(),
        Branch: fmt.Sprintf("%s:%s", branch, branch),
        Env:    repo_module.InternalPushingEnvironment(doer, forkRepo),
    })
}
```

#### Ж. CI/CD интеграция (автоматическое тестирование)

Файл: `internal/edu/notifier.go`

1. `EduNotifier` регистрируется через `notify.RegisterNotifier` при старте.
2. Когда Forgejo Actions runner завершает workflow, срабатывает `ActionRunNowDone`.
3. Наш нотификатор:
   - Ищет submission по `run.RepoID` (это форк студента).
   - Если нашёл — обновляет статус: `Success` → `passed`, `Failure` → `failed`.
   - Создаёт запись `TestResult` с `CommitSHA`, `Score` (0 или 100) и описанием.

#### З. Ручное оценивание (Grading)

Файлы: `routers/web/edu/grading.go`, `internal/edu/service_grading.go`, `templates/edu/submission_detail.tmpl`

1. На странице submissions (`/edu/teacher/assignments/{id}/submissions`) преподаватель видит таблицу со столбцами: Student, Status, CI Score, Grade, Repo, Updated, Detail.
2. Нажимает "Detail" → переходит на `/edu/teacher/assignments/{id}/submissions/{subID}`.
3. Видит: информацию о студенте, историю CI-прогонов (таблица TestResult), форму оценивания.
4. Вводит оценку (0–100) и комментарий, нажимает "Save Grade".
5. Оценка сохраняется в полях `grade`, `comment`, `graded_by_id`, `graded_unix` таблицы submissions.

Студент видит свою оценку на странице задания (`/edu/student/assignments/{id}`), а также последний результат CI.

#### И. Административная панель

Файлы: `routers/web/edu/admin.go`, `templates/edu/admin_panel.tmpl`

- `/edu/admin` — управление глобальными ролями пользователей (student/teacher/admin).
- Роли определяют, какие разделы видит пользователь (student vs teacher dashboards).

### 2.6 Полная карта роутов

```
/edu
├── /dashboard                          → Redirect по роли (teacher→assignments, student→assignments)
│
├── /student
│   ├── /assignments                    → Список заданий (только по enrolled курсам)
│   ├── /assignments/{id}               → Детали задания + CI результат + оценка
│   └── /assignments/{id}/join          → POST: взять задание (fork + submission)
│
├── /teacher
│   ├── /assignments                    → Список заданий преподавателя
│   ├── /assignments/new                → GET/POST: создать задание
│   ├── /assignments/{id}/edit          → GET/POST: редактировать задание
│   ├── /assignments/{id}/delete        → POST: удалить задание
│   ├── /assignments/{id}/submissions   → Таблица работ студентов
│   ├── /assignments/{id}/bulk-fork        → POST: массовый форк
│   ├── /assignments/{id}/bulk-fork-status → GET: статус/прогресс массового форка
│   ├── /assignments/{id}/sync-forks       → POST: синхронизация форков
│   ├── /assignments/{id}/sync-fork-status → GET: статус/прогресс синхронизации
│   ├── /assignments/{id}/submissions/{subID}       → Детали submission
│   ├── /assignments/{id}/submissions/{subID}/grade → POST: выставить оценку
│   ├── /dashboard                      → Redirect на /teacher/assignments
│   │
│   └── /courses
│       ├── /                           → Список курсов
│       ├── /new                        → GET/POST: создать курс
│       ├── /{id}                       → Детали курса (участники)
│       ├── /{id}/edit                  → GET/POST: редактировать курс
│       ├── /{id}/delete                → POST: удалить курс
│       ├── /{id}/enroll                → POST: записать студента
│       ├── /{id}/unenroll              → POST: отчислить студента
│       ├── /{id}/import                → GET/POST: загрузка CSV
│       ├── /{id}/import/{draftID}/preview    → GET: предпросмотр импорта
│       ├── /{id}/import/{draftID}/update-row → POST: правка строки
│       ├── /{id}/import/{draftID}/execute    → POST: выполнить импорт
│       └── /{id}/import/{draftID}/delete     → POST: удалить черновик
│
└── /admin
    ├── /                               → Панель управления ролями
    └── /roles                          → POST: обновить роль пользователя
```

### 2.7 Интеграция с ядром Forgejo

Edu-модуль затрагивает ядро Forgejo в **4 точках**:

| Точка | Файл ядра | Что делается |
|-------|-----------|-------------|
| **Init** | `routers/init.go` | `mustInitCtx(ctx, edu.Init)` — запуск инициализации |
| **Routes** | `routers/web/web.go` | `edu.RegisterRoutes(m, ...)` — регистрация роутов |
| **Notifier** | (runtime) | `notify.RegisterNotifier(&EduNotifier{})` — подписка на события |
| **Navbar** | `templates/custom/extra_links.tmpl` | Ссылка "Education" в навигации |

Весь остальной код изолирован в `internal/edu/`, `routers/web/edu/`, `templates/edu/`.

---

## Часть 3: Технический стек и паттерны

### 3.1 SQL: Squirrel (не Xorm для запросов)

Хотя Forgejo использует Xorm для ORM, edu-модуль использует **squirrel** (SQL builder) для всех запросов через `database/sql`:

```go
psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

query, args, err := psql.Select("id", "title", "course_id").
    From("edu_assignments").
    Where(sq.Eq{"course_id": courseID}).
    OrderBy("created_unix DESC").
    ToSql()

rows, err := r.runner.QueryContext(ctx, query, args...)
```

**Почему**: чистый SQL, безопасные плейсхолдеры, явный контроль над запросами. Xorm используется только для auto-migration схемы (`e.Sync()`).

### 3.2 Фронтенд: Server-side rendering

- **Fomantic UI** (форк Semantic UI) — CSS-фреймворк. Классы: `ui container`, `ui segment`, `ui celled table`, `ui label`, `ui form`.
- **jQuery** — минимально, для interactive компонентов.
- **Go templates** — `{{.Variable}}`, `{{range}}`, `{{if}}`, `{{template "base/head"}}`.
- **Встроенные хелперы**: `{{svg "octicon-check"}}` для иконок, `{{.CsrfTokenHtml}}` для CSRF, `{{(DateUtils).AbsoluteShort .Timestamp}}` для дат.
- **Приведение типов**: Go templates **не имеют** функции `string`. Для приведения кастомных типов (например, `SubmissionStatus`, `RoleType`) к строке используйте `{{printf "%s" .Value}}`.
- **Нет SPA**: каждая страница рендерится сервером.

### 3.3 Локализация (i18n)

Forgejo использует собственную систему интернационализации на основе INI-файлов с секциями.

#### Где хранятся переводы

Файлы локали находятся в `options/locale/`:
- `locale_en-US.ini` — английский (базовый, fallback)
- `locale_ru-RU.ini` — русский

Edu-ключи расположены в секции `[edu]` в конце каждого файла:

```ini
[edu]
assignments=Задания
new_assignment=Новое задание
no_assignments=Заданий не найдено.
roles=Роли
manage_roles=Управление ролями
...
```

INI-парсер автоматически формирует ключ как `секция.имя`: секция `[edu]` + ключ `roles` = `edu.roles`.

#### Использование в шаблонах

В Forgejo шаблонах локаль доступна через template function `ctx`, а **не** через `.locale`:

```html
<!-- ПРАВИЛЬНО — ctx это template function, возвращает templates.Context -->
{{ctx.Locale.Tr "edu.roles"}}

<!-- НЕПРАВИЛЬНО — .locale ищет ctx.Data["locale"], которого не существует -->
{{.locale.Tr "edu.roles"}}
```

`ctx.Locale.Tr` работает одинаково как внутри `{{range}}`, так и вне — потому что `ctx` это функция, а не поле данных. Внутри `{{range}}` точка (`.`) меняется на текущий элемент, но `ctx` по-прежнему доступен:

```html
{{range .Assignments}}
    <!-- . = текущий Assignment, но ctx.Locale всё ещё работает -->
    <span>{{ctx.Locale.Tr "edu.view"}}</span>
{{end}}
```

#### Как добавить новый ключ перевода

1. Добавить ключ в секцию `[edu]` файла `options/locale/locale_en-US.ini`:
   ```ini
   [edu]
   my_new_key=My new text
   ```
2. Добавить перевод в `options/locale/locale_ru-RU.ini`:
   ```ini
   [edu]
   my_new_key=Мой новый текст
   ```
3. Использовать в шаблоне: `{{ctx.Locale.Tr "edu.my_new_key"}}`
4. Использовать в Go-хендлере (для заголовков страниц и flash-сообщений): `ctx.Tr("edu.my_new_key")`

#### Как это работает под капотом

1. При старте `modules/translation/translation.go` → `InitLocales()` загружает все INI-файлы через `options.AssetFS()`.
2. `i18n/localestore.go` → `AddLocaleByIni()` парсит секции и строит карту `trKeyToIdxMap` (ключ → индекс) и `idxToMsgMap` (индекс → перевод).
3. Если ключ не найден в текущей локали, система fallback'ится на дефолтный язык (en-US). Если и там нет — возвращает имя ключа как есть (например, `"edu.roles"`).
4. В шаблонах `ctx` — это template function, зарегистрированная в `htmlrenderer.go`, которая возвращает `templates.Context` с полем `Locale`.

### 3.4 Локальная разработка (Docker)

Forgejo не компилируется на Windows (файлы `_unix.go` используют `syscall.Setpgid`, `unix.Umask` и т.д.). Для локальной разработки используется Docker.

**Файлы:**
- `Dockerfile.dev` — двухстадийная сборка: `golang:1.25-alpine` (build) → `alpine:3.23` (runtime). Запуск от пользователя `git` (Forgejo не работает от root).
- `docker-compose.dev.yml` — сервис `forgejo` (порт 3000), опциональные сервисы тестов (profile: test).
- `docker/app.ini` — преконфигурированный SQLite, `INSTALL_LOCK=true` (пропускает Install Wizard), Forgejo Actions включены.

**Запуск:**
```bash
cd forgejo-edu
docker compose -f docker-compose.dev.yml build forgejo
docker compose -f docker-compose.dev.yml up forgejo
# Сайт: http://localhost:3000, первый зарегистрированный пользователь — админ
```

**Сброс БД:**
```bash
docker compose -f docker-compose.dev.yml down -v   # -v удаляет volume с данными
docker compose -f docker-compose.dev.yml up forgejo --build
```

### 3.5 Тестирование

| Тип | Файлы | Как запускать |
|-----|-------|--------------|
| Unit-тесты (Go) | `internal/edu/*_test.go` (19 файлов) | `go test ./internal/edu/...` |
| Integration (Go + SQLite) | `tests/integration/edu_assignments_test.go` (15 тестов) | `make test-sqlite#TestEdu` |
| E2E (Playwright) | `tests/e2e/edu.test.e2e.ts` (8 тестов) | `make test-e2e-sqlite` |

**Integration тесты**: in-memory SQLite, fixtures из `models/fixtures/*.yml` (включая `user_role.yml` для edu ролей), хелперы `tests.PrepareTestEnv(t)()` и `loginUser(t, "user1")`. Тесты используют helper `ensureEduTables()` для создания edu-таблиц через DDL и `setupEduEnv()` для подготовки окружения.

**E2E тесты**: Playwright, SQLite-сервер на `localhost:3003`, fixture-юзеры (password="password").

**Docker-тесты:**
```bash
docker compose -f docker-compose.dev.yml run --rm test-integration   # Integration
docker compose -f docker-compose.dev.yml run --rm test-unit          # Unit
```

### 3.6 Ошибки и логирование

```go
// Паттерн ошибок
return fmt.Errorf("context: %w", err)

// Логирование
log.Error("EduNotifier: failed to create test result: %v", err)
log.Info("Educational Extension initialized successfully.")
```

### 3.7 Права доступа

В хендлерах проверка прав выполняется через Forgejo core:

```go
perm, err := access_model.GetUserRepoPermission(ctx, repo, ctx.Doer)
if !perm.IsAdmin() && !perm.CanWrite(unit_model.TypeCode) {
    ctx.Error(http.StatusForbidden, "Only instructors can view this page")
    return
}
```

---

## Часть 4: Как развивать проект

### Добавление новой сущности

1. Определить структуру в `models.go` (с тегами `xorm` и `json`). Добавить метод `TableName()` если squirrel-запросы будут обращаться к таблице с `edu_` префиксом.
2. Добавить `new(Entity)` в `e.Sync()` в `init.go`.
3. Добавить CRUD-методы в интерфейс `Repository` (`service.go`).
4. Реализовать методы в `repository_<entity>.go`.
5. Добавить бизнес-методы в `EducationalService` и реализовать в `service_<entity>.go`.
6. Создать хендлеры в `routers/web/edu/<entity>.go`.
7. Добавить роуты в `routes.go`.
8. Создать шаблоны в `templates/edu/<entity>_*.tmpl`.
9. Написать тесты.

### Добавление нового роута

1. Добавить строку в `routes.go` внутри нужной группы (`/student`, `/teacher`, `/admin`).
2. Создать хендлер в соответствующем файле.
3. Паттерн хендлера:
   ```go
   func MyHandler(ctx *context.Context) {
       svc := getEduService(ctx)
       if svc == nil {
           ctx.ServerError("getEduService", nil)
           return
       }
       // ... бизнес-логика ...
       ctx.Data["Key"] = value
       ctx.HTML(http.StatusOK, tplMyTemplate)
   }
   ```

### Шпаргалка

*   **Как добавить роут?** → `routers/web/edu/routes.go`
*   **Где модели?** → `internal/edu/models.go`
*   **Где логика?** → `internal/edu/service_*.go`
*   **Где SQL?** → `internal/edu/repository_*.go`
*   **Где шаблоны?** → `templates/edu/`
*   **Где мост к Forgejo?** → `internal/edu/adapter.go`
*   **Как зарегистрировать роуты в ядре?** → `routers/web/web.go` → `edu.RegisterRoutes`
*   **Как подписаться на события?** → `internal/edu/notifier.go` → `notify.RegisterNotifier`

Код модульный: изменения в `internal/edu/` не влияют на ядро Forgejo. Удачи!
