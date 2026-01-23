#!/bin/bash
set -e

# =============================================================================
# Forgejo Deployment Script
# Развертывание Forgejo на удаленной машине без Docker
# =============================================================================

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# =============================================================================
# Конфигурация - измени эти значения под свои нужды
# =============================================================================
PUBLIC_IP="${PUBLIC_IP:-$(curl -s ifconfig.me 2>/dev/null || hostname -I | awk '{print $1}')}"
HTTP_PORT="${HTTP_PORT:-3000}"
SSH_PORT="${SSH_PORT:-2222}"
APP_NAME="${APP_NAME:-Forgejo}"
RUN_USER="${RUN_USER:-git}"

# База данных: sqlite3 или postgres
DB_TYPE="${DB_TYPE:-postgres}"
DB_NAME="${DB_NAME:-forgejo}"
DB_USER="${DB_USER:-forgejo}"
DB_HOST="${DB_HOST:-127.0.0.1:5432}"

# Директории
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
FORGEJO_BINARY="${SCRIPT_DIR}/gitea"
DATA_DIR="${DATA_DIR:-/var/lib/forgejo}"
CONFIG_DIR="${CONFIG_DIR:-/etc/forgejo}"
LOG_DIR="${LOG_DIR:-/var/log/forgejo}"

# =============================================================================
# Проверка зависимостей
# =============================================================================
check_dependencies() {
    log_info "Проверка зависимостей..."
    
    local missing=()
    
    command -v git >/dev/null 2>&1 || missing+=("git")
    command -v sqlite3 >/dev/null 2>&1 || log_warn "sqlite3 не установлен (опционально для отладки)"
    
    if [[ ! -f "$FORGEJO_BINARY" ]]; then
        log_error "Бинарник Forgejo не найден: $FORGEJO_BINARY"
        log_info "Собираем из исходников..."
        build_forgejo
    fi
    
    if [[ ${#missing[@]} -gt 0 ]]; then
        log_error "Отсутствуют зависимости: ${missing[*]}"
        exit 1
    fi
    
    log_info "Все зависимости установлены"
}

# =============================================================================
# Сборка Forgejo (если бинарник отсутствует)
# =============================================================================
build_forgejo() {
    log_info "Сборка Forgejo..."
    cd "$SCRIPT_DIR"
    
    # Проверяем go и npm
    command -v go >/dev/null 2>&1 || { log_error "Go не установлен"; exit 1; }
    command -v npm >/dev/null 2>&1 || { log_error "npm не установлен"; exit 1; }
    
    # Устанавливаем npm зависимости и собираем фронтенд
    log_info "Установка npm зависимостей..."
    npm ci
    
    log_info "Сборка фронтенда..."
    make frontend
    
    log_info "Сборка бэкенда..."
    TAGS="bindata sqlite sqlite_unlock_notify" make backend
    
    log_info "Сборка завершена!"
}

# =============================================================================
# Установка и настройка PostgreSQL
# =============================================================================
setup_postgresql() {
    if [[ "$DB_TYPE" != "postgres" ]]; then
        return
    fi
    
    log_info "Настройка PostgreSQL..."
    
    # Проверяем установлен ли PostgreSQL
    if ! command -v psql &>/dev/null; then
        log_info "Установка PostgreSQL..."
        apt-get update
        apt-get install -y postgresql postgresql-contrib
    fi
    
    # Запускаем PostgreSQL
    systemctl start postgresql
    systemctl enable postgresql
    
    # Генерируем пароль если его нет
    if [[ ! -f "$CONFIG_DIR/.db_password" ]]; then
        DB_PASS=$(openssl rand -base64 24 | tr -d '\n/+=')
        echo "$DB_PASS" > "$CONFIG_DIR/.db_password"
        chmod 600 "$CONFIG_DIR/.db_password"
    fi
    DB_PASS=$(cat "$CONFIG_DIR/.db_password")
    
    # Создаем пользователя и базу данных
    if ! sudo -u postgres psql -tAc "SELECT 1 FROM pg_roles WHERE rolname='$DB_USER'" | grep -q 1; then
        log_info "Создание пользователя PostgreSQL: $DB_USER"
        sudo -u postgres psql -c "CREATE USER $DB_USER WITH PASSWORD '$DB_PASS';"
    fi
    
    if ! sudo -u postgres psql -tAc "SELECT 1 FROM pg_database WHERE datname='$DB_NAME'" | grep -q 1; then
        log_info "Создание базы данных: $DB_NAME"
        sudo -u postgres psql -c "CREATE DATABASE $DB_NAME OWNER $DB_USER ENCODING 'UTF8';"
        sudo -u postgres psql -c "GRANT ALL PRIVILEGES ON DATABASE $DB_NAME TO $DB_USER;"
    fi
    
    log_info "PostgreSQL настроен"
}

# =============================================================================
# Создание пользователя git (если не существует)
# =============================================================================
create_user() {
    if id "$RUN_USER" &>/dev/null; then
        log_info "Пользователь $RUN_USER уже существует"
    else
        log_info "Создание пользователя $RUN_USER..."
        useradd -r -m -d /home/$RUN_USER -s /bin/bash $RUN_USER
    fi
}

# =============================================================================
# Создание директорий
# =============================================================================
create_directories() {
    log_info "Создание директорий..."
    
    mkdir -p "$DATA_DIR"/{repositories,lfs,tmp,sessions,avatars,repo-avatars,attachments,indexers}
    mkdir -p "$CONFIG_DIR"
    mkdir -p "$LOG_DIR"
    
    chown -R $RUN_USER:$RUN_USER "$DATA_DIR"
    chown -R $RUN_USER:$RUN_USER "$CONFIG_DIR"
    chown -R $RUN_USER:$RUN_USER "$LOG_DIR"
    
    log_info "Директории созданы"
}

# =============================================================================
# Генерация конфигурации
# =============================================================================
generate_config() {
    local config_file="$CONFIG_DIR/app.ini"
    
    if [[ -f "$config_file" ]]; then
        log_warn "Конфигурация уже существует: $config_file"
        read -p "Перезаписать? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            return
        fi
    fi
    
    log_info "Генерация конфигурации..."
    
    # Генерируем секретный ключ
    SECRET_KEY=$(head -c 32 /dev/urandom | base64 | tr -d '\n')
    INTERNAL_TOKEN=$(head -c 64 /dev/urandom | base64 | tr -d '\n')
    
    cat > "$config_file" << EOF
; Автоматически сгенерировано скриптом edu_deploy.sh
; Документация: https://forgejo.org/docs/latest/admin/config-cheat-sheet/

APP_NAME = ${APP_NAME}
RUN_MODE = prod
RUN_USER = ${RUN_USER}

[server]
PROTOCOL = http
DOMAIN = ${PUBLIC_IP}
ROOT_URL = http://${PUBLIC_IP}:${HTTP_PORT}/
HTTP_ADDR = 0.0.0.0
HTTP_PORT = ${HTTP_PORT}
SSH_DOMAIN = ${PUBLIC_IP}
SSH_PORT = ${SSH_PORT}
SSH_LISTEN_PORT = ${SSH_PORT}
DISABLE_SSH = false
START_SSH_SERVER = true
LFS_START_SERVER = true
OFFLINE_MODE = false
APP_DATA_PATH = ${DATA_DIR}

$(if [[ "$DB_TYPE" == "postgres" ]]; then
    DB_PASS=$(cat "$CONFIG_DIR/.db_password" 2>/dev/null || echo "changeme")
    cat << DBEOF
[database]
DB_TYPE = postgres
HOST = ${DB_HOST}
NAME = ${DB_NAME}
USER = ${DB_USER}
PASSWD = ${DB_PASS}
SSL_MODE = disable
LOG_SQL = false
DBEOF
else
    cat << DBEOF
[database]
DB_TYPE = sqlite3
PATH = ${DATA_DIR}/forgejo.db
LOG_SQL = false
DBEOF
fi)

[repository]
ROOT = ${DATA_DIR}/repositories

[repository.local]
LOCAL_COPY_PATH = ${DATA_DIR}/tmp/local-repo

[repository.upload]
TEMP_PATH = ${DATA_DIR}/tmp/uploads

[indexer]
ISSUE_INDEXER_PATH = ${DATA_DIR}/indexers/issues.bleve

[session]
PROVIDER = file
PROVIDER_CONFIG = ${DATA_DIR}/sessions

[picture]
AVATAR_UPLOAD_PATH = ${DATA_DIR}/avatars
REPOSITORY_AVATAR_UPLOAD_PATH = ${DATA_DIR}/repo-avatars

[attachment]
PATH = ${DATA_DIR}/attachments

[log]
MODE = console, file
LEVEL = info
ROOT_PATH = ${LOG_DIR}

[security]
INSTALL_LOCK = false
SECRET_KEY = ${SECRET_KEY}
INTERNAL_TOKEN = ${INTERNAL_TOKEN}

[service]
DISABLE_REGISTRATION = false
REQUIRE_SIGNIN_VIEW = false
REGISTER_EMAIL_CONFIRM = false
ENABLE_NOTIFY_MAIL = false

[lfs]
PATH = ${DATA_DIR}/lfs

[mailer]
ENABLED = false

[cache]
ADAPTER = memory

[queue]
TYPE = level
DATADIR = ${DATA_DIR}/queues
EOF
    
    chown $RUN_USER:$RUN_USER "$config_file"
    chmod 640 "$config_file"
    
    log_info "Конфигурация сохранена: $config_file"
}

# =============================================================================
# Установка systemd сервиса
# =============================================================================
install_systemd_service() {
    log_info "Установка systemd сервиса..."
    
    # Копируем бинарник
    cp "$FORGEJO_BINARY" /usr/local/bin/forgejo
    chmod +x /usr/local/bin/forgejo
    
    # Создаем симлинк forgejo-cli
    ln -sf /usr/local/bin/forgejo /usr/local/bin/forgejo-cli
    
    # Определяем зависимость от базы данных
    local db_wants=""
    local db_after=""
    if [[ "$DB_TYPE" == "postgres" ]]; then
        db_wants="Wants=postgresql.service"
        db_after="After=postgresql.service"
    fi
    
    cat > /etc/systemd/system/forgejo.service << EOF
[Unit]
Description=Forgejo (Beyond coding. We forge.)
After=syslog.target
After=network.target
${db_after}
${db_wants}

[Service]
RestartSec=2s
Type=simple
User=${RUN_USER}
Group=${RUN_USER}
WorkingDirectory=${DATA_DIR}
ExecStart=/usr/local/bin/forgejo web --config ${CONFIG_DIR}/app.ini
Restart=always
Environment=USER=${RUN_USER} HOME=/home/${RUN_USER} FORGEJO_WORK_DIR=${DATA_DIR}

[Install]
WantedBy=multi-user.target
EOF
    
    systemctl daemon-reload
    log_info "Systemd сервис установлен"
}

# =============================================================================
# Запуск сервиса
# =============================================================================
start_service() {
    log_info "Запуск Forgejo..."
    
    systemctl enable forgejo
    systemctl start forgejo
    
    sleep 2
    
    if systemctl is-active --quiet forgejo; then
        log_info "Forgejo успешно запущен!"
    else
        log_error "Ошибка запуска Forgejo"
        systemctl status forgejo
        exit 1
    fi
}

# =============================================================================
# Настройка файрволла
# =============================================================================
setup_firewall() {
    log_info "Настройка файрволла..."
    
    if command -v ufw >/dev/null 2>&1; then
        ufw allow ${HTTP_PORT}/tcp comment 'Forgejo HTTP'
        ufw allow ${SSH_PORT}/tcp comment 'Forgejo SSH'
        log_info "UFW правила добавлены"
    elif command -v firewall-cmd >/dev/null 2>&1; then
        firewall-cmd --permanent --add-port=${HTTP_PORT}/tcp
        firewall-cmd --permanent --add-port=${SSH_PORT}/tcp
        firewall-cmd --reload
        log_info "Firewalld правила добавлены"
    else
        log_warn "Файрволл не найден. Убедитесь, что порты ${HTTP_PORT} и ${SSH_PORT} открыты"
    fi
}

# =============================================================================
# Вывод информации
# =============================================================================
print_info() {
    echo ""
    echo "=============================================="
    echo -e "${GREEN}Forgejo успешно развернут!${NC}"
    echo "=============================================="
    echo ""
    echo "🌐 Веб-интерфейс: http://${PUBLIC_IP}:${HTTP_PORT}"
    echo "🔑 SSH доступ:    ssh://git@${PUBLIC_IP}:${SSH_PORT}"
    echo ""
    echo "📁 Конфигурация:  ${CONFIG_DIR}/app.ini"
    echo "💾 Данные:        ${DATA_DIR}"
    echo "📋 Логи:          ${LOG_DIR}"
    echo ""
    echo "📝 Управление сервисом:"
    echo "   systemctl status forgejo"
    echo "   systemctl restart forgejo"
    echo "   journalctl -u forgejo -f"
    echo ""
    echo "⚠️  При первом входе создайте администратора!"
    echo "=============================================="
}

# =============================================================================
# Основная логика
# =============================================================================
main() {
    echo ""
    echo "=============================================="
    echo "  Развертывание Forgejo"
    echo "=============================================="
    echo ""
    
    # Проверка root
    if [[ $EUID -ne 0 ]]; then
        log_error "Скрипт должен запускаться от root"
        exit 1
    fi
    
    check_dependencies
    create_user
    create_directories
    setup_postgresql
    generate_config
    install_systemd_service
    start_service
    setup_firewall
    print_info
}

# =============================================================================
# CLI
# =============================================================================
case "${1:-deploy}" in
    deploy)
        main
        ;;
    build)
        build_forgejo
        ;;
    stop)
        systemctl stop forgejo
        log_info "Forgejo остановлен"
        ;;
    restart)
        systemctl restart forgejo
        log_info "Forgejo перезапущен"
        ;;
    status)
        systemctl status forgejo
        ;;
    logs)
        journalctl -u forgejo -f
        ;;
    *)
        echo "Использование: $0 {deploy|build|stop|restart|status|logs}"
        exit 1
        ;;
esac
