#!/usr/bin/env bash
# 「爱教学」Sprint 1 数据链路一键验收（后端 AGENTS.md §15）
#
# 前置：MySQL 已按 database/schema.sql + database/seed.sql 初始化，且已执行 make seed。
# 设计：只读校验（对照 MySQL 文档 §6 速查表）先执行，写操作（建课/上传）后执行，
#       避免写操作污染种子数据的期望值。
# 用法：BASE=http://localhost:8080/api/v1 bash scripts/verify.sh
set -uo pipefail

BASE="${BASE:-http://localhost:8080/api/v1}"
PASS=0
FAIL=0

c_green=$'\033[32m'; c_red=$'\033[31m'; c_dim=$'\033[2m'; c_reset=$'\033[0m'

pass() { PASS=$((PASS + 1)); printf '%s  PASS%s %s\n' "$c_green" "$c_reset" "$1"; }
fail() { FAIL=$((FAIL + 1)); printf '%s  FAIL%s %s\n' "$c_red" "$c_reset" "$1"; }

# json <json> <python 表达式>：表达式里的变量 d 为解析后的对象
json() { python3 -c 'import json,sys;d=json.load(sys.stdin);print(eval(sys.argv[1]))' "$2" <<<"$1" 2>/dev/null; }

check_eq() { # check_eq <描述> <期望> <实际>
  if [ "$2" = "$3" ]; then pass "$1（$3）"; else fail "$1：期望 $2，实际 $3"; fi
}

want() { # want <描述> <期望> <json> <表达式>
  local actual; actual="$(json "$3" "$4")"
  check_eq "$1" "$2" "$actual"
}

req() { curl -sS -m 15 "$@"; }

echo "${c_dim}== 0. 技术基线 S1.1 ==${c_reset}"
want "GET /healthz 返回 up" "up" "$(req "$BASE/healthz")" "d['data']['status']"

echo
echo "${c_dim}== 1. 三角色登录 S1.2 ==${c_reset}"
login() {
  req -X POST "$BASE/auth/login" -H 'Content-Type: application/json' \
    -d "{\"username\":\"$1\",\"password\":\"${2:-123456}\"}"
}
RESP_D="$(login director)";   TOK_D="$(json "$RESP_D" "d['data']['token']")"
RESP_T="$(login teacher)";    TOK_T="$(json "$RESP_T" "d['data']['token']")"
RESP_S="$(login supervisor)"; TOK_S="$(json "$RESP_S" "d['data']['token']")"

want "主任登录返回角色"   "director"   "$RESP_D" "d['data']['user']['role']"
want "教师登录返回角色"   "teacher"    "$RESP_T" "d['data']['user']['role']"
want "督导登录返回角色"   "supervisor" "$RESP_S" "d['data']['user']['role']"
want "主任返回教研室名称" "软件工程教研室" "$RESP_D" "d['data']['user']['department']"
want "错误密码返回 40101" "40101" "$(login director wrong-password)" "d['code']"
want "GET /auth/me 返回本人" "director" "$(req "$BASE/auth/me" -H "Authorization: Bearer $TOK_D")" "d['data']['role']"

AUTH_D=(-H "Authorization: Bearer $TOK_D")
AUTH_T=(-H "Authorization: Bearer $TOK_T")
AUTH_S=(-H "Authorization: Bearer $TOK_S")

echo
echo "${c_dim}== 2. 课程列表数据范围 S2.1/S2.2/S2.3 ==${c_reset}"
want "主任仅见本室课程 total=4" "4" "$(req "$BASE/courses?page=1&pageSize=10" "${AUTH_D[@]}")" "d['data']['total']"
want "教师仅见本人课程 total=2" "2" "$(req "$BASE/courses?page=1&pageSize=10" "${AUTH_T[@]}")" "d['data']['total']"
want "督导见全校课程 total=8"   "8" "$(req "$BASE/courses?page=1&pageSize=10" "${AUTH_S[@]}")" "d['data']['total']"
want "关键词搜索 SE3101 命中 1" "1" "$(req "$BASE/courses?keyword=SE3101" "${AUTH_S[@]}")" "d['data']['total']"
want "按教研室筛选 dept=1 命中 4" "4" "$(req "$BASE/courses?departmentId=1" "${AUTH_S[@]}")" "d['data']['total']"

echo
echo "${c_dim}== 3. 课程详情聚合 S4.1 ==${c_reset}"
DETAIL="$(req "$BASE/courses/1" "${AUTH_T[@]}")"
want "课程 1 班级数=2"     "2"  "$DETAIL" "d['data']['classCount']"
want "课程 1 学生人次=86"  "86" "$DETAIL" "d['data']['studentCount']"
want "课程 1 资源数=3"     "3"  "$DETAIL" "d['data']['resourceCount']"
want "课程 1 班级明细 2 条" "2"  "$DETAIL" "len(d['data']['classes'])"
want "课程 1 编码"          "SE3101" "$DETAIL" "d['data']['code']"

echo
echo "${c_dim}== 4. 督导覆盖率与听评课安排 S5.1（对照 MySQL 文档 §6）==${c_reset}"
COV="$(req "$BASE/supervision/coverage" "${AUTH_S[@]}")"
want "当前学期 open 课程数=6" "6"   "$COV" "d['data']['totalCourses']"
want "已完成听评课程数=3"     "3"   "$COV" "d['data']['supervisedCourses']"
want "总体覆盖率=0.5"         "0.5" "$COV" "d['data']['rate']"
want "分教研室覆盖率条目=3"   "3"   "$COV" "len(d['data']['byDepartment'])"
PLANS="$(req "$BASE/supervision/plans?page=1&pageSize=10" "${AUTH_S[@]}")"
want "听评课安排总数=5"       "5"   "$PLANS" "d['data']['total']"
want "筛选 status=completed 命中 3" "3" "$(req "$BASE/supervision/plans?status=completed" "${AUTH_S[@]}")" "d['data']['total']"

echo
echo "${c_dim}== 5. 工作台聚合（对照 MySQL 文档 §6 速查表）==${c_reset}"
DASH_D="$(req "$BASE/dashboard" "${AUTH_D[@]}")"
want "主任：本室课程数=4"   "4" "$DASH_D" "d['data']['courseCount']"
want "主任：本室教师数=3"   "3" "$DASH_D" "d['data']['teacherCount']"
want "主任：本学期班次=7"   "7" "$DASH_D" "d['data']['classCount']"
want "主任：本室资源数=4"   "4" "$DASH_D" "d['data']['resourceCount']"

DASH_T="$(req "$BASE/dashboard" "${AUTH_T[@]}")"
want "教师：我的课程数=2"   "2"   "$DASH_T" "d['data']['courseCount']"
want "教师：授课班级数=4"   "4"   "$DASH_T" "d['data']['classCount']"
want "教师：学生总人次=175" "175" "$DASH_T" "d['data']['studentCount']"
want "教师：资源总数=3"     "3"   "$DASH_T" "d['data']['resourceCount']"

DASH_S="$(req "$BASE/dashboard" "${AUTH_S[@]}")"
want "督导：open 课程数=6" "6"   "$DASH_S" "d['data']['courseCount']"
want "督导：计划数=5"       "5"   "$DASH_S" "d['data']['planCount']"
want "督导：已完成=3"       "3"   "$DASH_S" "d['data']['completedCount']"
want "督导：覆盖率=0.5"     "0.5" "$DASH_S" "d['data']['coverageRate']"

echo
echo "${c_dim}== 6. 字典接口 ==${c_reset}"
want "教研室数量=3" "3" "$(req "$BASE/departments" "${AUTH_D[@]}")" "len(d['data'])"
want "本室教师数=2" "2" "$(req "$BASE/teachers?departmentId=1" "${AUTH_D[@]}")" "len(d['data'])"

echo
echo "${c_dim}== 7. 越权负向用例 ==${c_reset}"
want "教师建课返回 40301"          "40301" "$(req -X POST "$BASE/courses" "${AUTH_T[@]}" -H 'Content-Type: application/json' -d '{}')" "d['code']"
want "主任访问督导接口返回 40301"  "40301" "$(req "$BASE/supervision/coverage" "${AUTH_D[@]}")" "d['code']"
want "教师查看他人课程返回 40302"  "40302" "$(req "$BASE/courses/3" "${AUTH_T[@]}")" "d['code']"
want "未登录访问返回 40101"        "40101" "$(req "$BASE/courses")" "d['code']"
want "伪造 token 返回 40101"       "40101" "$(req "$BASE/courses" -H 'Authorization: Bearer forged.token.value')" "d['code']"

echo
echo "${c_dim}== 8. 主任新增/修改课程 S3.1（写操作）==${c_reset}"
NEW_CODE="SE$((RANDOM % 9000 + 1000))"
CREATE="$(req -X POST "$BASE/courses" "${AUTH_D[@]}" -H 'Content-Type: application/json' \
  -d "{\"code\":\"$NEW_CODE\",\"name\":\"验收课程\",\"departmentId\":1,\"teacherId\":2,\"semester\":\"2026-2027-1\",\"credit\":2,\"hours\":32,\"description\":\"verify.sh\",\"status\":\"open\"}")"
NEW_ID="$(json "$CREATE" "d['data']['id']")"
want "新增课程成功" "0" "$CREATE" "d['code']"
want "新增后主任课程数=5" "5" "$(req "$BASE/courses" "${AUTH_D[@]}")" "d['data']['total']"

DUP="$(req -X POST "$BASE/courses" "${AUTH_D[@]}" -H 'Content-Type: application/json' \
  -d "{\"code\":\"$NEW_CODE\",\"name\":\"重复课程\",\"departmentId\":1,\"teacherId\":2,\"semester\":\"2026-2027-1\",\"credit\":2,\"hours\":32,\"status\":\"open\"}")"
want "重复编码+学期返回 40901" "40901" "$DUP" "d['code']"

BADCODE="$(req -X POST "$BASE/courses" "${AUTH_D[@]}" -H 'Content-Type: application/json' \
  -d '{"code":"bad-code","name":"非法编码","departmentId":1,"teacherId":2,"semester":"2026-2027-1","credit":2,"hours":32,"status":"open"}')"
want "非法编码返回 40001" "40001" "$BADCODE" "d['code']"

if [ -n "$NEW_ID" ]; then
  UPD="$(req -X PUT "$BASE/courses/$NEW_ID" "${AUTH_D[@]}" -H 'Content-Type: application/json' \
    -d "{\"code\":\"$NEW_CODE\",\"name\":\"验收课程（已改）\",\"departmentId\":1,\"teacherId\":3,\"semester\":\"2026-2027-1\",\"credit\":3,\"hours\":48,\"description\":\"verify.sh\",\"status\":\"draft\"}")"
  want "修改课程成功"         "0"          "$UPD" "d['code']"
  want "修改后课程名称生效"   "验收课程（已改）" "$UPD" "d['data']['name']"
  want "修改后 code 不可被改" "$NEW_CODE"  "$UPD" "d['data']['code']"
  want "修改后状态=draft"     "draft"      "$UPD" "d['data']['status']"
fi

echo
echo "${c_dim}== 9. 教师上传/删除资源 S4.2（写操作）==${c_reset}"
TMPFILE="$(mktemp /tmp/aijiaoxue-verify-XXXXXX.pdf)"
printf '%%PDF-1.4\n1 0 obj\n<< /Type /Catalog >>\nendobj\ntrailer\n<< /Root 1 0 R >>\n%%%%EOF\n' >"$TMPFILE"
UP="$(req -X POST "$BASE/courses/1/resources" "${AUTH_T[@]}" -F "file=@$TMPFILE")"
want "教师上传资源成功"     "0"   "$UP" "d['code']"
want "上传资源类型为 pdf"   "pdf" "$UP" "d['data']['type']"
want "上传资源上传人=李明"  "李明" "$UP" "d['data']['uploader']"
RES_ID="$(json "$UP" "d['data']['id']")"
want "上传后课程 1 资源数=4" "4" "$(req "$BASE/courses/1/resources" "${AUTH_T[@]}")" "len(d['data'])"
want "非法扩展名返回 40001" "40001" "$(req -X POST "$BASE/courses/1/resources" "${AUTH_T[@]}" -F "file=@/etc/hostname")" "d['code']"

if [ -n "$RES_ID" ]; then
  want "教师删除资源成功" "0" "$(req -X DELETE "$BASE/resources/$RES_ID" "${AUTH_T[@]}")" "d['code']"
  want "删除后课程 1 资源数=3" "3" "$(req "$BASE/courses/1/resources" "${AUTH_T[@]}")" "len(d['data'])"
fi

# 清理本次验收新增的课程，保持种子数据纯净
if [ -n "$NEW_ID" ]; then
  mysql -h 127.0.0.1 -P "${MYSQL_PORT:-3306}" -u aijiaoxue -paijiaoxue_dev aijiaoxue \
    -e "DELETE FROM courses WHERE id=$NEW_ID;" >/dev/null 2>&1 \
    && echo "${c_dim}已清理验收课程 id=$NEW_ID${c_reset}" \
    || echo "${c_dim}请手工清理验收课程：DELETE FROM courses WHERE id=$NEW_ID;${c_reset}"
fi

rm -f "$TMPFILE"

echo
printf '验收结果：%s%d 通过%s / %s%d 失败%s\n' "$c_green" "$PASS" "$c_reset" "$c_red" "$FAIL" "$c_reset"
[ "$FAIL" -eq 0 ]
