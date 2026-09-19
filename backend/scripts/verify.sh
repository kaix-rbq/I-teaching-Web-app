#!/usr/bin/env bash
# 「爱教学」Sprint 1 + Sprint 2.1 数据链路一键验收（后端 AGENTS.md §15）
#
# 前置：MySQL 已按 database/schema.sql + migrations（V2）+ database/seed.sql 初始化，
#       且已执行 make seed。Sprint 2.1 断言对照开发计划 §3.4 对账基准与 §4.2 响应示例。
# 设计：只读校验先执行，写操作（建课/上传/建场次/提交评分）后执行，
#       避免写操作污染种子数据的期望值；写操作产生的新增数据在脚本尾部清理。
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

echo
echo "${c_dim}== 10. Sprint 2.1 授课记录与双侧评分摘要（对照开发计划 §3.4 对账基准）==${c_reset}"
C1SESS="$(req "$BASE/courses/1/sessions?page=1&pageSize=10" "${AUTH_S[@]}")"
want "课程 1 授课记录总数=3"          "3"     "$C1SESS" "d['data']['total']"
want "最新一场主题=迭代计划与估点"     "迭代计划与估点" "$C1SESS" "d['data']['list'][0]['topic']"
want "最新一场督导摘要分=82.5"        "82.5"  "$C1SESS" "d['data']['list'][0]['supervisorScore']"
want "最新一场智能体摘要分=75"        "75"    "$C1SESS" "d['data']['list'][0]['agentScore']"
want "最早一场督导摘要分=52.5"        "52.5"  "$C1SESS" "d['data']['list'][2]['supervisorScore']"
want "最早一场智能体摘要分=60.71"     "60.71" "$C1SESS" "d['data']['list'][2]['agentScore']"
want "最早一场评价条数=2"             "2"     "$C1SESS" "d['data']['list'][2]['evaluationCount']"
want "学期切片不匹配返回空"           "0"     "$(req "$BASE/courses/1/sessions?semester=2025-2026-2" "${AUTH_S[@]}")" "d['data']['total']"

S1="$(req "$BASE/sessions/1" "${AUTH_S[@]}")"
want "场次 1 课程名=软件项目管理"      "软件项目管理" "$S1" "d['data']['courseName']"
want "场次 1 教师名=李明"             "李明"   "$S1" "d['data']['teacherName']"
want "场次 1 状态=evaluated"          "evaluated" "$S1" "d['data']['status']"

E1="$(req "$BASE/sessions/1/evaluation" "${AUTH_S[@]}")"
want "场次 1 督导评分条数=1"           "1"     "$E1" "len(d['data']['supervisorScores'])"
want "场次 1 督导评分人=陈静"          "陈静"   "$E1" "d['data']['supervisorScores'][0]['evaluatorName']"
want "场次 1 督导 objective=4"        "4"     "$E1" "d['data']['supervisorScores'][0]['objective']"
want "场次 1 督导总分=52.5"           "52.5"  "$E1" "d['data']['supervisorScores'][0]['totalScore']"
want "场次 1 智能体参考=qwen-audio-v1" "qwen-audio-v1" "$E1" "d['data']['agentScore']['aiModelVersion']"
want "场次 1 智能体 objective=null"   "None"  "$E1" "d['data']['agentScore']['objective']"
want "场次 1 智能体置信度=0.72"        "0.72"  "$E1" "d['data']['agentScore']['aiConfidence']"

echo
echo "${c_dim}== 11. 教师级/课程级聚合（§4.2 响应示例逐位对账）==${c_reset}"
TS="$(req "$BASE/teachers/2/evaluation-summary" "${AUTH_D[@]}")"
want "李明综合分=70.42"                "70.42" "$TS" "d['data']['compositeScore']"
want "李明督导侧=68.33"               "68.33" "$TS" "d['data']['supervisorScore']"
want "李明智能体侧=67.86"             "67.86" "$TS" "d['data']['agentScore']"
want "objective 维度分=83.33"         "83.33" "$TS" "[x['score'] for x in d['data']['dimensions'] if x['key']=='objective'][0]"
want "objective 无智能体分"           "None"  "$TS" "[x['agentScore'] for x in d['data']['dimensions'] if x['key']=='objective'][0]"
want "interaction 维度分=54.17"       "54.17" "$TS" "[x['score'] for x in d['data']['dimensions'] if x['key']=='interaction'][0]"
want "frontier 维度分=50"             "50"    "$TS" "[x['score'] for x in d['data']['dimensions'] if x['key']=='frontier'][0]"
want "frontier 权重=0"                "0"     "$TS" "[x['weight'] for x in d['data']['dimensions'] if x['key']=='frontier'][0]"
want "frontier 标记为观测项"          "True"  "$TS" "[x['isObservation'] for x in d['data']['dimensions'] if x['key']=='frontier'][0]"
want "样本量场次=3"                   "3"     "$TS" "d['data']['sample']['sessionCount']"
want "双侧对齐场次=3"                 "3"     "$TS" "d['data']['sample']['alignedCount']"
want "样本充足"                       "True"  "$TS" "d['data']['sample']['sampleSufficient']"
want "无 flags"                       "[]"    "$TS" "d['data']['flags']"
want "计分口径=v1"                    "v1"    "$TS" "d['data']['formulaVersion']"
want "按课程明细 1 条"                "1"     "$TS" "len(d['data']['courses'])"

CS="$(req "$BASE/courses/1/evaluation-summary" "${AUTH_T[@]}")"
want "课程级综合分=70.42（与教师级逐位一致）" "70.42" "$CS" "d['data']['compositeScore']"
want "课程级督导侧=68.33"             "68.33" "$CS" "d['data']['supervisorScore']"
want "课程级样本场次=3"               "3"     "$CS" "d['data']['sample']['sessionCount']"

echo
echo "${c_dim}== 12. 教师评分列表与权限矩阵（§5.1）==${c_reset}"
want "主任见本室教师评分 total=2"      "2"     "$(req "$BASE/teacher-scores" "${AUTH_D[@]}")" "d['data']['total']"
want "督导见全校教师评分 total=4"      "4"     "$(req "$BASE/teacher-scores" "${AUTH_S[@]}")" "d['data']['total']"
want "督导按教研室筛选 dept=2 命中 1"  "1"     "$(req "$BASE/teacher-scores?departmentId=2" "${AUTH_S[@]}")" "d['data']['total']"
want "列表中李明综合分=70.42"          "70.42" "$(req "$BASE/teacher-scores" "${AUTH_D[@]}")" "[x['compositeScore'] for x in d['data']['list'] if x['teacherName']=='李明'][0]"
want "未知学期返回 40001"             "40001" "$(req "$BASE/teacher-scores?semester=2099-2100-1" "${AUTH_S[@]}")" "d['code']"
want "教师访问评分列表返回 40301"     "40301" "$(req "$BASE/teacher-scores" "${AUTH_T[@]}")" "d['code']"
want "李明查本人面板=70.42"           "70.42" "$(req "$BASE/teachers/2/evaluation-summary" "${AUTH_T[@]}")" "d['data']['compositeScore']"
want "教师查同事面板返回 40302"       "40302" "$(req "$BASE/teachers/3/evaluation-summary" "${AUTH_T[@]}")" "d['code']"
ZH="$(req "$BASE/teachers/3/evaluation-summary" "${AUTH_D[@]}")"
want "张华无评分 flags 含 no_data"     "True"  "$ZH" "'no_data' in d['data']['flags']"
want "张华综合分为 null"              "None"  "$ZH" "d['data']['compositeScore']"
want "教师查他室课程授课记录 40302"   "40302" "$(req "$BASE/courses/3/sessions" "${AUTH_T[@]}")" "d['code']"
want "主任查他室课程授课记录 40302"   "40302" "$(req "$BASE/courses/3/sessions" "${AUTH_D[@]}")" "d['code']"

echo
echo "${c_dim}== 13. 督导创建授课记录与评分提交（写操作，S6.1/S6.3）==${c_reset}"
want "主任创建授课记录返回 40301"     "40301" "$(req -X POST "$BASE/sessions" "${AUTH_D[@]}" -H 'Content-Type: application/json' -d '{"courseId":1,"sessionDate":"2026-09-16","period":"验收节次"}')" "d['code']"
want "未来日期返回 40002"             "40002" "$(req -X POST "$BASE/sessions" "${AUTH_S[@]}" -H 'Content-Type: application/json' -d '{"courseId":1,"sessionDate":"2099-01-01","period":"1-2 节"}')" "d['code']"
want "课程不存在返回 40401"           "40401" "$(req -X POST "$BASE/sessions" "${AUTH_S[@]}" -H 'Content-Type: application/json' -d '{"courseId":9999,"sessionDate":"2026-09-16","period":"1-2 节"}')" "d['code']"
want "同日同节次重复返回 40901"       "40901" "$(req -X POST "$BASE/sessions" "${AUTH_S[@]}" -H 'Content-Type: application/json' -d '{"courseId":1,"sessionDate":"2026-09-12","period":"3-4 节"}')" "d['code']"

SESS="$(req -X POST "$BASE/sessions" "${AUTH_S[@]}" -H 'Content-Type: application/json' \
  -d '{"courseId":1,"sessionDate":"2026-09-16","period":"验收节次","topic":"验收脚本创建"}')"
SESS_ID="$(json "$SESS" "d['data']['id']")"
want "督导创建授课记录成功"           "0"        "$SESS" "d['code']"
want "新场次初始状态=scheduled"       "scheduled" "$SESS" "d['data']['status']"
want "新场次教师随课程落库=李明"      "李明"      "$SESS" "d['data']['teacherName']"

EV="$(req -X PUT "$BASE/sessions/$SESS_ID/supervisor-evaluation" "${AUTH_S[@]}" -H 'Content-Type: application/json' \
  -d '{"objective":5,"content":5,"interaction":4,"organization":5,"frontier":4,"comment":"verify.sh"}')"
want "督导提交评分成功"               "0"   "$EV" "d['code']"
want "单次总分=95"                    "95"  "$EV" "d['data']['totalScore']"
want "评分写入口径版本=v1"            "v1"  "$EV" "d['data']['formulaVersion']"
want "评分后场次状态=evaluated"       "evaluated" "$(req "$BASE/sessions/$SESS_ID" "${AUTH_S[@]}")" "d['data']['status']"
want "空维度提交返回 40002"           "40002" "$(req -X PUT "$BASE/sessions/$SESS_ID/supervisor-evaluation" "${AUTH_S[@]}" -H 'Content-Type: application/json' -d '{"comment":"无维度"}')" "d['code']"
want "维度越界返回 40001"             "40001" "$(req -X PUT "$BASE/sessions/$SESS_ID/supervisor-evaluation" "${AUTH_S[@]}" -H 'Content-Type: application/json' -d '{"objective":6}')" "d['code']"
want "教师提交督导评分返回 40301"     "40301" "$(req -X PUT "$BASE/sessions/$SESS_ID/supervisor-evaluation" "${AUTH_T[@]}" -H 'Content-Type: application/json' -d '{"objective":5}')" "d['code']"

EV2="$(req -X PUT "$BASE/sessions/$SESS_ID/supervisor-evaluation" "${AUTH_S[@]}" -H 'Content-Type: application/json' \
  -d '{"objective":5,"content":5,"interaction":4,"organization":5,"frontier":4,"comment":"verify.sh 重提"}')"
want "重复提交幂等覆盖总分不变"       "95"  "$EV2" "d['data']['totalScore']"
want "幂等覆盖后评价仍为 1 条"        "1"   "$(req "$BASE/sessions/$SESS_ID/evaluation" "${AUTH_S[@]}")" "len(d['data']['supervisorScores'])"
want "新增后课程 1 授课记录总数=4"    "4"   "$(req "$BASE/courses/1/sessions" "${AUTH_S[@]}")" "d['data']['total']"

rm -f "$TMPFILE"

# 清理本次验收新增的课程，保持种子数据纯净
if [ -n "$NEW_ID" ]; then
  mysql -h 127.0.0.1 -P "${MYSQL_PORT:-3306}" -u aijiaoxue -paijiaoxue_dev aijiaoxue \
    -e "DELETE FROM courses WHERE id=$NEW_ID;" >/dev/null 2>&1 \
    && echo "${c_dim}已清理验收课程 id=$NEW_ID${c_reset}" \
    || echo "${c_dim}请手工清理验收课程：DELETE FROM courses WHERE id=$NEW_ID;${c_reset}"
fi

# 清理本次验收新增的授课记录与评分，保持 §3.4 对账基准纯净
if [ -n "$SESS_ID" ]; then
  mysql -h 127.0.0.1 -P "${MYSQL_PORT:-3306}" -u aijiaoxue -paijiaoxue_dev aijiaoxue \
    -e "DELETE FROM evaluations WHERE session_id=$SESS_ID; DELETE FROM teaching_sessions WHERE id=$SESS_ID;" >/dev/null 2>&1 \
    && echo "${c_dim}已清理验收场次 id=$SESS_ID${c_reset}" \
    || echo "${c_dim}请手工清理验收场次：DELETE FROM evaluations WHERE session_id=$SESS_ID; DELETE FROM teaching_sessions WHERE id=$SESS_ID;${c_reset}"
fi

echo
printf '验收结果：%s%d 通过%s / %s%d 失败%s\n' "$c_green" "$PASS" "$c_reset" "$c_red" "$FAIL" "$c_reset"
[ "$FAIL" -eq 0 ]
