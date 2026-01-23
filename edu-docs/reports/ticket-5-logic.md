# Отчет по задаче №5: Логика управления заданиями

## Что сделано
1. **Расширение Repository**:
   - В интерфейс `SQLRunner` добавлен метод `QueryContext`.
   - Реализован метод `GetAssignments` в `internal/edu/repository_ext.go` для выборки списка заданий.

2. **Расширение Service**:
   - В интерфейсы `EducationalService` и `Repository` добавлен метод `GetAssignments`.
   - Реализована прокси-логика в `internal/edu/service_ext.go`.

3. **Тестирование**:
   - Написаны Unit-тесты для метода `GetAssignments` в сервисе (`internal/edu/service_ext_test.go`).
   - Тесты проходят успешно.

## Как верифицировать
1. Запустить тесты:
   ```bash
   go test -v ./internal/edu/...
   ```
2. Убедиться, что новые тесты `TestGetAssignments_Service` проходят.
