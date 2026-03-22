#!/bin/bash
# ============================================================================
# edu_merge_upstream.sh — Безопасный мерж из upstream Forgejo
#
# Скрипт автоматизирует слияние изменений из основного репозитория Forgejo
# в форк с образовательным расширением. Автоматически резолвит тривиальные
# конфликты (go.sum, go.mod) и проверяет наличие edu-строк в файлах ядра.
#
# Использование:
#   ./edu_merge_upstream.sh [upstream-branch]
#   upstream-branch — ветка upstream для мержа (по умолчанию: forgejo)
#
# Примеры:
#   ./edu_merge_upstream.sh            # мерж из upstream/forgejo
#   ./edu_merge_upstream.sh v12.0      # мерж из upstream/v12.0
# ============================================================================

set -euo pipefail

UPSTREAM_BRANCH="${1:-forgejo}"
UPSTREAM_REMOTE="upstream"
UPSTREAM_URL="https://codeberg.org/forgejo/forgejo.git"

# Edu integration points — строки, которые ДОЛЖНЫ быть в файлах ядра
EDU_IMPORT_INIT='"forgejo.org/internal/edu"'
EDU_CALL_INIT='mustInitCtx(ctx, edu.Init)'
EDU_IMPORT_WEB='"forgejo.org/routers/web/edu"'
EDU_CALL_WEB='edu.RegisterRoutes(m, reqSignIn)'

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

info()  { echo -e "${GREEN}[INFO]${NC} $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC} $*"; }
error() { echo -e "${RED}[ERROR]${NC} $*"; }
die()   { error "$*"; exit 1; }

# ── Предварительные проверки ──────────────────────────────────────────────

# Проверяем что мы в git-репозитории
git rev-parse --git-dir > /dev/null 2>&1 || die "Не git-репозиторий. Запустите из forgejo-edu/"

# Проверяем чистоту рабочей директории
if [ -n "$(git status --porcelain)" ]; then
    die "Рабочая директория не чистая. Закоммитьте или stash-ните изменения."
fi

CURRENT_BRANCH=$(git branch --show-current)
info "Текущая ветка: $CURRENT_BRANCH"

# ── Настройка upstream remote ────────────────────────────────────────────

if ! git remote get-url "$UPSTREAM_REMOTE" > /dev/null 2>&1; then
    info "Добавляю upstream remote: $UPSTREAM_URL"
    git remote add "$UPSTREAM_REMOTE" "$UPSTREAM_URL"
else
    info "Upstream remote уже настроен: $(git remote get-url $UPSTREAM_REMOTE)"
fi

# ── Fetch upstream ───────────────────────────────────────────────────────

info "Получаю изменения из upstream/$UPSTREAM_BRANCH..."
git fetch "$UPSTREAM_REMOTE" "$UPSTREAM_BRANCH" || die "Не удалось получить upstream/$UPSTREAM_BRANCH"

# Показываем сколько коммитов позади
BEHIND=$(git rev-list --count HEAD.."$UPSTREAM_REMOTE/$UPSTREAM_BRANCH" 2>/dev/null || echo "?")
info "Коммитов для мержа: $BEHIND"

if [ "$BEHIND" = "0" ]; then
    info "Уже актуально, мерж не нужен."
    exit 0
fi

# ── Создание ветки для мержа ─────────────────────────────────────────────

MERGE_BRANCH="merge-upstream-$(date +%Y-%m-%d)"
info "Создаю ветку: $MERGE_BRANCH"
git checkout -b "$MERGE_BRANCH" || die "Не удалось создать ветку $MERGE_BRANCH"

# ── Мерж ─────────────────────────────────────────────────────────────────

info "Выполняю мерж upstream/$UPSTREAM_BRANCH..."
MERGE_RESULT=0
git merge "$UPSTREAM_REMOTE/$UPSTREAM_BRANCH" --no-edit || MERGE_RESULT=$?

if [ "$MERGE_RESULT" -eq 0 ]; then
    info "Мерж прошёл без конфликтов!"
else
    info "Есть конфликты, пробую авторезолв..."

    # Получаем список конфликтующих файлов
    CONFLICTS=$(git diff --name-only --diff-filter=U)
    UNRESOLVED=""

    for file in $CONFLICTS; do
        case "$file" in
            go.sum)
                info "  Резолвлю go.sum (go mod tidy)..."
                git checkout --theirs go.sum 2>/dev/null || true
                go mod tidy 2>/dev/null
                if [ $? -eq 0 ]; then
                    git add go.sum
                    info "  ✓ go.sum — готово"
                else
                    warn "  go mod tidy завершился с ошибкой"
                    UNRESOLVED="$UNRESOLVED $file"
                fi
                ;;
            go.mod)
                info "  Резолвлю go.mod (go mod tidy)..."
                git checkout --theirs go.mod 2>/dev/null || true
                go mod tidy 2>/dev/null
                if [ $? -eq 0 ]; then
                    git add go.mod
                    info "  ✓ go.mod — готово"
                else
                    warn "  go mod tidy завершился с ошибкой"
                    UNRESOLVED="$UNRESOLVED $file"
                fi
                ;;
            *)
                warn "  $file — требует ручного резолва"
                UNRESOLVED="$UNRESOLVED $file"
                ;;
        esac
    done

    if [ -n "$UNRESOLVED" ]; then
        echo ""
        error "Не удалось автоматически разрешить конфликты в:"
        for f in $UNRESOLVED; do
            echo "  - $f"
        done
        echo ""
        warn "Разрешите конфликты вручную, затем выполните:"
        echo "  git add <файлы>"
        echo "  git commit"
        echo "  # Затем запустите скрипт повторно для проверки edu-строк:"
        echo "  $0 --check-only"
        exit 1
    fi

    # Все конфликты разрешены — завершаем мерж
    git commit --no-edit || die "Не удалось завершить мерж-коммит"
    info "Все конфликты разрешены автоматически."
fi

# ── Проверка edu-строк в файлах ядра ─────────────────────────────────────

info "Проверяю наличие edu-интеграции в файлах ядра..."
EDU_OK=true

check_line() {
    local file="$1"
    local pattern="$2"
    local description="$3"

    if ! grep -qF "$pattern" "$file" 2>/dev/null; then
        error "  ОТСУТСТВУЕТ в $file: $description"
        error "    Ожидалось: $pattern"
        EDU_OK=false
    else
        info "  ✓ $file: $description"
    fi
}

check_line "routers/init.go"    "$EDU_IMPORT_INIT"  "import edu"
check_line "routers/init.go"    "$EDU_CALL_INIT"    "mustInitCtx edu.Init"
check_line "routers/web/web.go" "$EDU_IMPORT_WEB"   "import edu routes"
check_line "routers/web/web.go" "$EDU_CALL_WEB"     "RegisterRoutes"

if [ "$EDU_OK" = false ]; then
    echo ""
    error "Edu-интеграция потеряна при мерже!"
    error "Добавьте недостающие строки вручную (см. документацию в edu-docs/developer_deep_dive.md, раздел 5.1)"
    exit 1
fi

# ── Сборка и тесты ──────────────────────────────────────────────────────

echo ""
info "Мерж завершён успешно. Ветка: $MERGE_BRANCH"
echo ""
info "Следующие шаги:"
echo "  1. Собрать проект:        make build"
echo "  2. Запустить edu-тесты:   make test-sqlite#TestEdu"
echo "  3. Создать PR:            git push -u origin $MERGE_BRANCH"
echo ""
info "Для отката:  git checkout $CURRENT_BRANCH && git branch -D $MERGE_BRANCH"
