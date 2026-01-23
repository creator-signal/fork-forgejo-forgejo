# Отчет по задаче №4: Модель данных и DAL

## Что сделано
1. **Репозиторий**:
   - Реализована структура `dbRepository` в `internal/edu/repository.go`.
   - Использован билдер `squirrel` для построения безопасных SQL-запросов.
   - Реализованы методы `CreateAssignment` и `GetAssignmentByID`.

2. **Тестирование**:
   - Написаны Unit-тесты в `internal/edu/repository_test.go` с использованием `go-sqlmock`.
   - Тесты покрывают успешные сценарии и обработку ошибок.
   - Исправлена проблема с генерацией SQL (пробелы в `VALUES`), тесты проходят.

## Как верифицировать
1. Запустить тесты:
   ```bash
   go test -v ./internal/edu/...
   ```
2. Ожидаемый результат:
   ```
   === RUN   TestCreateAssignment_Repo
   --- PASS: TestCreateAssignment_Repo (0.00s)
   === RUN   TestGetAssignmentByID_Repo
   --- PASS: TestGetAssignmentByID_Repo (0.00s)
   PASS
   ```
