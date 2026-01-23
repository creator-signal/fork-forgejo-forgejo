# Руководство по развёртыванию Forgejo

Это руководство описывает полный процесс развёртывания Forgejo на удалённом сервере с Ubuntu 22.04.

## Требования

- **ОС**: Ubuntu 22.04 LTS
- **RAM**: минимум 1 GB (рекомендуется 2 GB)
- **Диск**: минимум 10 GB свободного места
- **Доступ**: root или sudo

## Быстрый старт

```bash
# Клонируем репозиторий
git clone https://github.com/your-org/forgejo-edu.git
cd forgejo-edu

# Запускаем развёртывание
sudo ./edu_edu_deploy.sh deploy
```

## Подробная инструкция

### 1. Подготовка сервера

```bash
# Обновляем систему
sudo apt update && sudo apt upgrade -y

# Устанавливаем необходимые пакеты
sudo apt install -y git curl wget
```

### 2. Установка Go (если нужна сборка из исходников)

```bash
# Скачиваем Go 1.22+
wget https://go.dev/dl/go1.22.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.22.0.linux-amd64.tar.gz

# Добавляем в PATH
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# Проверяем
go version
```

### 3. Установка Node.js (если нужна сборка из исходников)

```bash
# Устанавливаем Node.js 20.x
curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
sudo apt install -y nodejs

# Проверяем
node --version
npm --version
```

### 4. Клонирование и сборка

```bash
# Клонируем репозиторий
git clone https://github.com/your-org/forgejo-edu.git
cd forgejo-edu

# Если бинарник отсутствует — собираем
./edu_deploy.sh build
```

### 5. Развёртывание

```bash
# Полное развёртывание (PostgreSQL + Forgejo + nginx)
sudo ./edu_edu_deploy.sh deploy
```

Скрипт автоматически:
- Создаст пользователя `git`
- Установит и настроит PostgreSQL
- Сгенерирует конфигурацию
- Установит systemd сервис
- Настроит nginx как reverse proxy

## Конфигурация

### Переменные окружения

Вы можете настроить развёртывание через переменные окружения:

| Переменная | По умолчанию | Описание |
|------------|--------------|----------|
| `PUBLIC_IP` | автоопределение | Публичный IP сервера |
| `HTTP_PORT` | `3000` | Порт Forgejo (внутренний) |
| `SSH_PORT` | `2222` | Порт SSH для Git |
| `APP_NAME` | `Forgejo` | Название приложения |
| `DB_TYPE` | `postgres` | Тип БД: `postgres` или `sqlite3` |
| `DB_NAME` | `forgejo` | Имя базы данных |
| `DB_USER` | `forgejo` | Пользователь БД |
| `DB_HOST` | `127.0.0.1:5432` | Хост PostgreSQL |

Пример с кастомными настройками:

```bash
sudo PUBLIC_IP=my.domain.com DB_TYPE=sqlite3 ./edu_deploy.sh deploy
```

### Файлы конфигурации

| Файл | Описание |
|------|----------|
| `/etc/forgejo/app.ini` | Основная конфигурация Forgejo |
| `/etc/forgejo/.db_password` | Пароль PostgreSQL |
| `/etc/nginx/sites-available/forgejo` | Конфигурация nginx |
| `/etc/systemd/system/forgejo.service` | Systemd юнит |

## Структура директорий

```
/var/lib/forgejo/          # Данные Forgejo
├── repositories/          # Git репозитории
├── lfs/                   # Git LFS объекты
├── avatars/               # Аватары пользователей
├── attachments/           # Вложения к issues
├── sessions/              # Сессии пользователей
└── indexers/              # Поисковые индексы

/var/log/forgejo/          # Логи
/etc/forgejo/              # Конфигурация
```

## Управление сервисом

```bash
# Статус
sudo systemctl status forgejo
sudo systemctl status postgresql
sudo systemctl status nginx

# Перезапуск
sudo systemctl restart forgejo

# Логи
sudo journalctl -u forgejo -f

# Или через скрипт
./edu_deploy.sh status
./edu_deploy.sh restart
./edu_deploy.sh logs
```

## Настройка nginx (reverse proxy)

Если порт 3000 закрыт провайдером, используйте nginx:

```bash
# Установка nginx
sudo apt install -y nginx

# Конфигурация уже создаётся автоматически в:
# /etc/nginx/sites-available/forgejo
```

Пример конфигурации nginx:

```nginx
server {
    listen 80;
    server_name your-domain.com;

    client_max_body_size 100M;

    location / {
        proxy_pass http://127.0.0.1:3000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # WebSocket support
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```

## SSL/HTTPS (опционально)

Для настройки HTTPS с Let's Encrypt:

```bash
# Установка certbot
sudo apt install -y certbot python3-certbot-nginx

# Получение сертификата
sudo certbot --nginx -d your-domain.com

# Автообновление (добавляется автоматически)
sudo systemctl enable certbot.timer
```

После этого обновите `ROOT_URL` в `/etc/forgejo/app.ini`:

```ini
ROOT_URL = https://your-domain.com/
```

## Первоначальная настройка

1. Откройте в браузере: `http://YOUR_IP/`
2. На странице установки настройте:
   - **Тип БД**: PostgreSQL (уже настроен)
   - **Хост БД**: `127.0.0.1:5432`
   - **Имя БД**: `forgejo`
   - **Пользователь**: `forgejo`
   - **Пароль**: см. `/etc/forgejo/.db_password`
3. Создайте администратора
4. Нажмите "Установить Forgejo"

## Резервное копирование

```bash
# Бэкап базы данных
sudo -u postgres pg_dump forgejo > backup_$(date +%Y%m%d).sql

# Бэкап данных
sudo tar -czf forgejo_data_$(date +%Y%m%d).tar.gz /var/lib/forgejo

# Бэкап конфигурации
sudo tar -czf forgejo_config_$(date +%Y%m%d).tar.gz /etc/forgejo
```

## Восстановление

```bash
# Восстановление БД
sudo -u postgres psql forgejo < backup_20240101.sql

# Восстановление данных
sudo tar -xzf forgejo_data_20240101.tar.gz -C /

# Перезапуск
sudo systemctl restart forgejo
```

## Обновление

```bash
cd /path/to/forgejo-edu
git pull

# Пересборка (если нужно)
./edu_deploy.sh build

# Обновление бинарника
sudo cp gitea /usr/local/bin/forgejo
sudo systemctl restart forgejo
```

## Устранение неполадок

### Порт недоступен извне

Если порт 3000 закрыт провайдером:
1. Используйте nginx на порту 80
2. Или измените `HTTP_PORT` на открытый порт

### Ошибка подключения к PostgreSQL

```bash
# Проверка статуса
sudo systemctl status postgresql

# Проверка подключения
sudo -u postgres psql -c "\l"

# Сброс пароля
NEW_PASS=$(openssl rand -base64 24)
sudo -u postgres psql -c "ALTER USER forgejo WITH PASSWORD '$NEW_PASS';"
echo "$NEW_PASS" | sudo tee /etc/forgejo/.db_password
```

### Логи для диагностики

```bash
# Forgejo
sudo journalctl -u forgejo -n 100

# PostgreSQL
sudo journalctl -u postgresql -n 50

# nginx
sudo tail -f /var/log/nginx/error.log
```

## Полезные ссылки

- [Документация Forgejo](https://forgejo.org/docs/latest/)
- [Настройка app.ini](https://forgejo.org/docs/latest/admin/config-cheat-sheet/)
- [Миграция с Gitea/GitHub](https://forgejo.org/docs/latest/admin/migrations/)

## Команды edu_deploy.sh

| Команда | Описание |
|---------|----------|
| `./edu_deploy.sh deploy` | Полное развёртывание |
| `./edu_deploy.sh build` | Сборка из исходников |
| `./edu_deploy.sh start` | Запуск сервиса |
| `./edu_deploy.sh stop` | Остановка сервиса |
| `./edu_deploy.sh restart` | Перезапуск сервиса |
| `./edu_deploy.sh status` | Статус сервиса |
| `./edu_deploy.sh logs` | Просмотр логов |
