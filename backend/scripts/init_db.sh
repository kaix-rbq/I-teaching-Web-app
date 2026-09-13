#!/usr/bin/env bash
# 一次性初始化数据库：建库建表 → 种子数据 → 写入 bcrypt 密码哈希。
# 用法：
#   bash scripts/init_db.sh                        # 使用默认 root 交互输密码
#   MYSQL_ADMIN_USER=root MYSQL_ADMIN_PASSWORD=*** bash scripts/init_db.sh
set -euo pipefail

cd "$(dirname "$0")/.."

MYSQL_ADMIN_USER="${MYSQL_ADMIN_USER:-root}"
MYSQL_ADMIN_PASSWORD="${MYSQL_ADMIN_PASSWORD:-}"
MYSQL_HOST="${MYSQL_HOST:-127.0.0.1}"
MYSQL_PORT="${MYSQL_PORT:-3306}"
DB_NAME="${DB_NAME:-aijiaoxue}"

mysql_admin() {
  if [ -n "$MYSQL_ADMIN_PASSWORD" ]; then
    mysql -h "$MYSQL_HOST" -P "$MYSQL_PORT" -u "$MYSQL_ADMIN_USER" -p"$MYSQL_ADMIN_PASSWORD" "$@"
  else
    mysql -h "$MYSQL_HOST" -P "$MYSQL_PORT" -u "$MYSQL_ADMIN_USER" -p "$@"
  fi
}

echo "==> [1/3] 建库建表：database/schema.sql"
mysql_admin < database/schema.sql

echo "==> [2/3] 写入种子数据：database/seed.sql"
mysql_admin "$DB_NAME" < database/seed.sql

echo "==> [3/3] 覆写演示账号 bcrypt 密码哈希：go run ./cmd/seed"
go run ./cmd/seed -config "${CONFIG:-config.yaml}"

echo
echo "==> 自查（期望 departments=3 users=6 courses=8 classes=14 resources=6 plans=5）"
mysql_admin "$DB_NAME" <<'SQL'
SELECT
  (SELECT COUNT(*) FROM departments)       AS departments,
  (SELECT COUNT(*) FROM users)             AS users,
  (SELECT COUNT(*) FROM courses)           AS courses,
  (SELECT COUNT(*) FROM course_classes)    AS classes,
  (SELECT COUNT(*) FROM resources)         AS resources,
  (SELECT COUNT(*) FROM supervision_plans) AS plans;
SQL
