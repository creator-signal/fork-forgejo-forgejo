# Отчет по задаче №6: Логика инициализации репозиториев

## Что сделано
- Реализованы методы репозитория `CreateSubmission` и `GetSubmission` для работы с сущностью `Submission` (попытка сдачи).
- Реализован метод сервиса `JoinAssignment`:
    - Проверяет наличие существующей попытки (Submission).
    - Если попытки нет, инициирует форк репозитория шаблона (Template Repository) в репозиторий студента.
    - Регистрирует новую попытку в БД.
- Внедрен интерфейс `RepoForker` для абстракции логики форка (взаимодействие с `services/repository`).

## Технические детали
- **Service Layer**: `JoinAssignment` оркестрирует процесс: проверка БД -> форк (через Forker) -> запись в БД.
- **Mocking**: Использована библиотека `testify/mock` для изоляции логики сервиса от БД и Git-операций.
- **Repository Interface**: Расширен методами для работы с Submissions.
- **WSL Adaptation**: Тесты и разработка переведены в WSL окружение для корректной работы с Unix-сигналами Forgejo.

## Тестирование
- Unit-тесты для `dbRepository` (`repository_submissions_test.go`): проверен SQL для Insert/Select.
- Unit-тесты для `EducationalService` (`service_join_test.go`):
    - Сценарий **Success**: успешный форк и создание записи.
    - Сценарий **AlreadyJoined**: корректный возврат существующей записи без дублирования форка.
- Регрессионное тестирование: все тесты пакета `internal/edu` пройдены успешно.

## Файлы
- `internal/edu/repository_submissions.go`
- `internal/edu/service_join.go`
- `internal/edu/service.go` (обновление интерфейсов)
- `internal/edu/repository_submissions_test.go`
- `internal/edu/service_join_test.go`
