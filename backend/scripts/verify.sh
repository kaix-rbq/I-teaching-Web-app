#!/usr/bin/env bash
# 「爱教学」Sprint 1 + Sprint 2.1 数据链路一键验收（后端 AGENTS.md §15）
#
# 前置：MySQL 已按 database/schema.sql + migrations（V1–V6）+ database/seed.sql 初始化，
#       且已执行 make seed。断言对照开发计划 §3.4 对账基准（§4 为契约索引，正文见 backend_AGENTS.md §8）；
#       聚合期望值与 pkg/scoring 的种子数据集单测同源（教师2=71.25 / 教师3=79.17 / c1=69.17）。
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
echo "${c_dim}== 4. 听评课安排 S5.1（覆盖率口径已停用 D2）==${c_reset}"
# 覆盖率不再作为业务指标：仅保留 deprecated 路由的 code smoke，不断言数值。
want "覆盖率接口保留 deprecated（仅 code smoke）" "0" "$(req "$BASE/supervision/coverage" "${AUTH_S[@]}")" "d['code']"
PLANS="$(req "$BASE/supervision/plans?page=1&pageSize=10" "${AUTH_S[@]}")"
want "听评课安排总数=7"       "7"   "$PLANS" "d['data']['total']"
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
want "督导：听评课安排=7"       "7"    "$DASH_S" "len(d['data']['recentPlans'])"
want "督导：最近草稿=0"         "0"    "$DASH_S" "len(d['data']['recentDrafts'])"
want "督导：待 AI 评析授课记录=0" "0"  "$DASH_S" "len(d['data']['pendingSessions'])"
want "督导：计划 1 已关联评估"   "True" "$DASH_S" "[p['evaluated'] for p in d['data']['recentPlans'] if p['id']==1][0]"

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
want "最新一场督导摘要分=null（计分只按AI）" "None" "$C1SESS" "d['data']['list'][0]['supervisorScore']"
want "最新一场智能体摘要分=75"        "75"    "$C1SESS" "d['data']['list'][0]['agentScore']"
want "最早一场督导摘要分=null（计分只按AI）" "None" "$C1SESS" "d['data']['list'][2]['supervisorScore']"
want "最早一场智能体摘要分=65"        "65"    "$C1SESS" "d['data']['list'][2]['agentScore']"
want "最早一场评价条数=2"             "2"     "$C1SESS" "d['data']['list'][2]['evaluationCount']"
want "学期切片不匹配返回空"           "0"     "$(req "$BASE/courses/1/sessions?semester=2025-2026-2" "${AUTH_S[@]}")" "d['data']['total']"

S1="$(req "$BASE/sessions/1" "${AUTH_S[@]}")"
want "场次 1 课程名=软件项目管理"      "软件项目管理" "$S1" "d['data']['courseName']"
want "场次 1 教师名=李明"             "李明"   "$S1" "d['data']['teacherName']"
want "场次 1 状态=evaluated"          "evaluated" "$S1" "d['data']['status']"

E1="$(req "$BASE/sessions/1/evaluation" "${AUTH_S[@]}")"
want "场次 1 督导评分条数=1"           "1"     "$E1" "len(d['data']['supervisorScores'])"
want "场次 1 督导评分人=陈静"          "陈静"   "$E1" "d['data']['supervisorScores'][0]['evaluatorName']"
want "场次 1 督导 objective=null（B8 不打分）" "None" "$E1" "d['data']['supervisorScores'][0]['objective']"
want "场次 1 督导总分=null（B8 不打分）"       "None" "$E1" "d['data']['supervisorScores'][0]['totalScore']"
want "场次 1 智能体参考=qwen-audio-v1" "qwen-audio-v1" "$E1" "d['data']['agentScore']['aiModelVersion']"
want "场次 1 智能体 objective=4"    "4"     "$E1" "d['data']['agentScore']['objective']"
want "场次 1 智能体置信度=0.7"         "0.7"   "$E1" "d['data']['agentScore']['aiConfidence']"
want "P1a 授课记录列表含 supervisorEvaluated" "True" "$C1SESS" "all('supervisorEvaluated' in x for x in d['data']['list'])"
want "P1a 评估聚合 supervisorEvaluated=true" "True" "$E1" "d['data']['supervisorEvaluated']"

echo
echo "${c_dim}== 10.1 P2 提优读接口 smoke ==${c_reset}"
AS="$(req "$BASE/courses/1/agent-suggestions?semester=2026-2027-1" "${AUTH_S[@]}")"
want "P2 课程提优建议 code=0"           "0"           "$AS" "d['code']"
want "P2 课程提优建议返回数组"          "True"        "$AS" "isinstance(d['data'], list)"
want "P2 课程提优建议按课次倒序"        "2026-09-26"  "$AS" "d['data'][0]['sessionDate']"
want "P2 课程提优建议 evidence 为字符串" "True"       "$AS" "isinstance(d['data'][0]['evidence'], str)"
TR="$(req "$BASE/teachers/2/score-trend?semester=2026-2027-1" "${AUTH_S[@]}")"
want "P2 教师趋势 code=0"               "0"           "$TR" "d['code']"
want "P2 教师趋势返回 6 条 series"      "6"           "$TR" "len(d['data'])"
want "P2 教师趋势首条=composite"        "composite"   "$TR" "d['data'][0]['key']"
want "P2 教师趋势 frontier 为观测项"    "True"        "$TR" "d['data'][5]['isObservation']"
want "P2 教师趋势点含 value 字段"       "True"        "$TR" "all('value' in p for s in d['data'] for p in s['points'])"
# 课程 2 属于种子教师 3，不属于 teacher 2，用于验证课程范围裁剪。
want "P6 教师访问他人课程建议 40302"    "40302"      "$(req "$BASE/courses/2/agent-suggestions?semester=2026-2027-1" "${AUTH_T[@]}")" "d['code']"
TR_DIM="$(req "$BASE/teachers/2/score-trend?semester=2026-2027-1&dimension=interaction" "${AUTH_S[@]}")"
want "P7 趋势 dimension 过滤只返回 interaction" "interaction" "$TR_DIM" "d['data'][0]['key']"
want "P7 趋势 dimension 过滤返回 1 条"           "1"           "$TR_DIM" "len(d['data'])"

echo
echo "${c_dim}== 10.2 P3 教师级 AI 报告 smoke（不要求真实模型生成）==${c_reset}"
# 建议以 AIJIAOXUE_AGENT_REPORT_ENABLED=false 运行本脚本，避免 regenerate 触发真实 S2 调用。
AR="$(req "$BASE/teachers/2/ai-report?semester=2026-2027-1" "${AUTH_S[@]}")"
want "P3 AI 报告读接口 code=0"          "0"     "$AR" "d['code']"
want "P3 AI 报告响应含 status"           "True"  "$AR" "'status' in d['data']"
want "P3 AI 报告响应含 report 字段"      "True"  "$AR" "'report' in d['data']"
want "P3 AI 报告 status 合法"            "True"  "$AR" "d['data']['status'] in ['pending','running','done','failed']"
REG="$(req -X POST "$BASE/teachers/2/ai-report/regenerate?semester=2026-2027-1" "${AUTH_S[@]}")"
want "P3 督导重生成返回 pending"         "pending" "$REG" "d['data']['status']"
AR_T="$(req "$BASE/teachers/2/ai-report?semester=2026-2027-1" "${AUTH_T[@]}")"
want "P7 教师本人读取 AI 报告"           "0"     "$AR_T" "d['code']"
want "P7 教师本人读取报告 status 合法"    "True"  "$AR_T" "d['data']['status'] in ['pending','running','done','failed']"
AR_D="$(req "$BASE/teachers/2/ai-report?semester=2026-2027-1" "${AUTH_D[@]}")"
want "P7 主任读取本室教师报告"           "0"     "$AR_D" "d['code']"
want "P3 教师重生成返回 40301"           "40301" "$(req -X POST "$BASE/teachers/2/ai-report/regenerate?semester=2026-2027-1" "${AUTH_T[@]}")" "d['code']"
want "P3 教师查看他人报告返回 40302"     "40302" "$(req "$BASE/teachers/3/ai-report?semester=2026-2027-1" "${AUTH_T[@]}")" "d['code']"

echo
echo "${c_dim}== 10.3 P4 SSE 错误帧 smoke（不触发模型）==${c_reset}"
chat_err="$(req -X POST "$BASE/agent/chat" "${AUTH_T[@]}" -H 'Content-Type: application/json' \
  -d '{"courseId":0,"question":"参数校验","history":[]}')"
if [[ "$chat_err" == *'"error"'* && "$chat_err" == *'"code":40001'* && "$chat_err" == *'data: [DONE]'* ]]; then
  pass "P4 SSE 错误帧结构可解析（40001 + [DONE]）"
else
  fail "P4 SSE 错误帧结构异常：$chat_err"
fi

echo
echo "${c_dim}== 11. 教师级/课程级聚合（与 pkg/scoring 种子数据集单测同源）==${c_reset}"
TS="$(req "$BASE/teachers/2/evaluation-summary" "${AUTH_D[@]}")"
want "李明综合分=72.5（只按AI）"       "72.5"  "$TS" "d['data']['compositeScore']"
want "李明督导侧=null（督导不计分）"    "None"  "$TS" "d['data']['supervisorScore']"
want "李明智能体侧=72.5"              "72.5"  "$TS" "d['data']['agentScore']"
want "objective 维度分=75"            "75"    "$TS" "[x['score'] for x in d['data']['dimensions'] if x['key']=='objective'][0]"
want "objective 智能体分=75"          "75"    "$TS" "[x['agentScore'] for x in d['data']['dimensions'] if x['key']=='objective'][0]"
want "interaction 维度分=60"          "60"    "$TS" "[x['score'] for x in d['data']['dimensions'] if x['key']=='interaction'][0]"
want "frontier 维度分=60"             "60"    "$TS" "[x['score'] for x in d['data']['dimensions'] if x['key']=='frontier'][0]"
want "frontier 权重=0"                "0"     "$TS" "[x['weight'] for x in d['data']['dimensions'] if x['key']=='frontier'][0]"
want "frontier 标记为观测项"          "True"  "$TS" "[x['isObservation'] for x in d['data']['dimensions'] if x['key']=='frontier'][0]"
want "样本量场次=5"                   "5"     "$TS" "d['data']['sample']['sessionCount']"
want "有AI计分数据场次=5"              "5"     "$TS" "d['data']['sample']['alignedCount']"
want "样本充足"                       "True"  "$TS" "d['data']['sample']['sampleSufficient']"
want "无 flags"                       "[]"    "$TS" "d['data']['flags']"
want "计分口径=v1"                    "v1"    "$TS" "d['data']['formulaVersion']"
want "按课程明细 2 条"                "2"     "$TS" "len(d['data']['courses'])"

CS="$(req "$BASE/courses/1/evaluation-summary" "${AUTH_T[@]}")"
want "课程级综合分=70（只按AI）" "70" "$CS" "d['data']['compositeScore']"
want "课程级督导侧=null"             "None" "$CS" "d['data']['supervisorScore']"
want "课程级样本场次=3"               "3"     "$CS" "d['data']['sample']['sessionCount']"

echo
echo "${c_dim}== 12. 教师评分列表与权限矩阵（§5.1）==${c_reset}"
want "主任见本室教师评分 total=2"      "2"     "$(req "$BASE/teacher-scores" "${AUTH_D[@]}")" "d['data']['total']"
want "督导见全校教师评分 total=4"      "4"     "$(req "$BASE/teacher-scores" "${AUTH_S[@]}")" "d['data']['total']"
want "督导按教研室筛选 dept=2 命中 1"  "1"     "$(req "$BASE/teacher-scores?departmentId=2" "${AUTH_S[@]}")" "d['data']['total']"
want "列表中李明综合分=72.5"          "72.5" "$(req "$BASE/teacher-scores" "${AUTH_D[@]}")" "[x['compositeScore'] for x in d['data']['list'] if x['teacherName']=='李明'][0]"
want "未知学期返回 40001"             "40001" "$(req "$BASE/teacher-scores?semester=2099-2100-1" "${AUTH_S[@]}")" "d['code']"
want "教师访问评分列表返回 40301"     "40301" "$(req "$BASE/teacher-scores" "${AUTH_T[@]}")" "d['code']"
want "李明查本人面板=72.5"             "72.5" "$(req "$BASE/teachers/2/evaluation-summary" "${AUTH_T[@]}")" "d['data']['compositeScore']"
want "教师查同事面板返回 40302"       "40302" "$(req "$BASE/teachers/3/evaluation-summary" "${AUTH_T[@]}")" "d['code']"
ZH="$(req "$BASE/teachers/3/evaluation-summary" "${AUTH_D[@]}")"
want "张华综合分=75.83"               "75.83" "$ZH" "d['data']['compositeScore']"
ZN="$(req "$BASE/teachers/2/evaluation-summary?semester=2025-2026-2" "${AUTH_D[@]}")"
want "旧学期无数据 flags 含 no_data"   "True"  "$ZN" "'no_data' in d['data']['flags']"
want "旧学期综合分为 null"             "None"  "$ZN" "d['data']['compositeScore']"
want "教师查他室课程授课记录 40302"   "40302" "$(req "$BASE/courses/3/sessions" "${AUTH_T[@]}")" "d['code']"
want "主任查他室课程授课记录 40302"   "40302" "$(req "$BASE/courses/3/sessions" "${AUTH_D[@]}")" "d['code']"

echo
echo "${c_dim}== 12.1 教师历次评价时间线（GET /teachers/:id/evaluations，T1.13）==${c_reset}"
TL="$(req "$BASE/teachers/2/evaluations?page=1&pageSize=10" "${AUTH_D[@]}")"
want "时间线场次总数=5"                  "5"            "$TL" "d['data']['total']"
want "按课次倒序（最新=2026-09-26）"      "2026-09-26"   "$TL" "d['data']['list'][0]['sessionDate']"
want "最新一行场次 id=3"                 "3"            "$TL" "d['data']['list'][0]['sessionId']"
want "最新一行课程名=软件项目管理"        "软件项目管理"  "$TL" "d['data']['list'][0]['courseName']"
want "最新一行课次主题=迭代计划与估点"     "迭代计划与估点" "$TL" "d['data']['list'][0]['topic']"
want "最新一行场次综合分=75（只按AI）"    "75"           "$TL" "d['data']['list'][0]['compositeScore']"
want "最新一行督导分=null（督导不计分）"   "None"         "$TL" "d['data']['list'][0]['supervisorScore']"
want "最新一行智能体分=75"               "75"           "$TL" "d['data']['list'][0]['agentScore']"
want "含督导评语 comment"                "估点练习设计巧妙，学生参与度高。" "$TL" "d['data']['list'][0]['supervisorEvaluations'][0]['comment']"
want "含督导亮点 highlights"             "练习设计贴近实战" "$TL" "d['data']['list'][0]['supervisorEvaluations'][0]['highlights']"
want "含督导建议 suggestions"            "预留 5 分钟总结" "$TL" "d['data']['list'][0]['supervisorEvaluations'][0]['suggestions']"
want "含督导待改进 improvements"         "时间略紧"      "$TL" "d['data']['list'][0]['supervisorEvaluations'][0]['improvements']"
want "含智能体模型=qwen-audio-v1"        "qwen-audio-v1" "$TL" "d['data']['list'][0]['agentEvaluation']['aiModelVersion']"
want "最早一行场次日期=2026-09-12"        "2026-09-12"   "$TL" "d['data']['list'][4]['sessionDate']"
want "最早一行场次综合分=65（只按AI）"    "65"           "$TL" "d['data']['list'][4]['compositeScore']"
TLP="$(req "$BASE/teachers/2/evaluations?page=1&pageSize=2" "${AUTH_D[@]}")"
want "分页 pageSize=2 → total 仍为 5"    "5"            "$TLP" "d['data']['total']"
want "分页 pageSize=2 → 本页 2 条"       "2"            "$TLP" "len(d['data']['list'])"
want "教师查本人时间线可用（最新 id=3）"  "3"            "$(req "$BASE/teachers/2/evaluations" "${AUTH_T[@]}")" "d['data']['list'][0]['sessionId']"
want "教师查同事时间线 40302"            "40302"        "$(req "$BASE/teachers/3/evaluations" "${AUTH_T[@]}")" "d['code']"
want "主任查他室教师时间线 40302"         "40302"        "$(req "$BASE/teachers/4/evaluations" "${AUTH_D[@]}")" "d['code']"
want "时间线未知学期 40001"              "40001"        "$(req "$BASE/teachers/2/evaluations?semester=2099-2100-1" "${AUTH_D[@]}")" "d['code']"

echo
echo "${c_dim}== 12.2 督导抽样任务队列（R1.2 / B2·T1）==${c_reset}"
AT="$(req "$BASE/supervision/ai-tasks" "${AUTH_S[@]}")"
want "抽样任务队列可用"                 "0"      "$AT" "d['code']"
want "抽样任务总数=1（李明满5条AI终评）" "1"      "$AT" "d['data']['total']"
want "抽样命中最低 AI 分场次=1"          "1"      "$AT" "d['data']['list'][0]['sessionId']"
want "抽样原因=lowest"                  "lowest" "$AT" "d['data']['list'][0]['reason']"
want "首批批号=1"                       "1"      "$AT" "d['data']['list'][0]['batch']"
want "已有督导观察标记 reviewed"        "True"   "$AT" "d['data']['list'][0]['reviewed']"
want "pending=1 过滤后为空（已复评）"   "0"      "$(req "$BASE/supervision/ai-tasks?pending=1" "${AUTH_S[@]}")" "d['data']['total']"
want "教师访问抽样队列 40301"           "40301"  "$(req "$BASE/supervision/ai-tasks" "${AUTH_T[@]}")" "d['code']"

echo
echo "${c_dim}== 13. 督导创建授课记录与评分提交（写操作，S6.1/S6.3）==${c_reset}"
want "主任创建授课记录返回 40301"     "40301" "$(req -X POST "$BASE/sessions" "${AUTH_D[@]}" -H 'Content-Type: application/json' -d '{"courseId":1,"classId":1,"sessionDate":"2026-09-16","period":"验收节次"}')" "d['code']"
want "缺 classId 返回 40002"           "40002" "$(req -X POST "$BASE/sessions" "${AUTH_S[@]}" -H 'Content-Type: application/json' -d '{"courseId":1,"sessionDate":"2026-09-16","period":"验收节次"}')" "d['code']"
want "未来日期返回 40002"             "40002" "$(req -X POST "$BASE/sessions" "${AUTH_S[@]}" -H 'Content-Type: application/json' -d '{"courseId":1,"classId":1,"sessionDate":"2099-01-01","period":"1-2 节"}')" "d['code']"
want "课程不存在返回 40401"           "40401" "$(req -X POST "$BASE/sessions" "${AUTH_S[@]}" -H 'Content-Type: application/json' -d '{"courseId":9999,"classId":1,"sessionDate":"2026-09-16","period":"1-2 节"}')" "d['code']"

SESS="$(req -X POST "$BASE/sessions" "${AUTH_S[@]}" -H 'Content-Type: application/json' \
  -d '{"courseId":1,"classId":1,"sessionDate":"2026-09-16","period":"验收节次","topic":"验收脚本创建"}')"
SESS_ID="$(json "$SESS" "d['data']['id']")"
want "督导创建授课记录成功"           "0"        "$SESS" "d['code']"
want "新场次初始状态=scheduled"       "scheduled" "$SESS" "d['data']['status']"
want "新场次教师随课程落库=李明"      "李明"      "$SESS" "d['data']['teacherName']"
TR_PENDING="$(req "$BASE/teachers/2/score-trend?semester=2026-2027-1" "${AUTH_S[@]}")"
want "P2 趋势无 agent 评价点为 null"   "True"      "$TR_PENDING" "any(p['value'] is None for s in d['data'] if s['key']=='composite' for p in s['points'] if p['date']=='2026-09-16')"
want "同日同节次重复返回 40901"       "40901" "$(req -X POST "$BASE/sessions" "${AUTH_S[@]}" -H 'Content-Type: application/json' -d '{"courseId":1,"classId":1,"sessionDate":"2026-09-16","period":"验收节次"}')" "d['code']"

EV="$(req -X PUT "$BASE/sessions/$SESS_ID/supervisor-observation" "${AUTH_S[@]}" -H 'Content-Type: application/json' \
  -d '{"comment":"verify.sh 审核意见","highlights":"亮点A","improvements":"主要问题B","suggestions":"改进建议C"}')"
want "督导提交观察成功"               "0"   "$EV" "d['code']"
want "观察无五维 objective=null"       "None" "$EV" "d['data']['objective']"
want "观察无五维 totalScore=null"      "None" "$EV" "d['data']['totalScore']"
want "观察 highlights 生效"            "亮点A" "$EV" "d['data']['highlights']"
want "观察写入口径版本=v1"             "v1"  "$EV" "d['data']['formulaVersion']"
want "督导观察不改场次状态（仍 scheduled）" "scheduled" "$(req "$BASE/sessions/$SESS_ID" "${AUTH_S[@]}")" "d['data']['status']"
want "空观察返回 40002"                "40002" "$(req -X PUT "$BASE/sessions/$SESS_ID/supervisor-observation" "${AUTH_S[@]}" -H 'Content-Type: application/json' -d '{}')" "d['code']"
want "教师提交督导观察返回 40301"      "40301" "$(req -X PUT "$BASE/sessions/$SESS_ID/supervisor-observation" "${AUTH_T[@]}" -H 'Content-Type: application/json' -d '{"comment":"x"}')" "d['code']"
want "旧五维评分接口返回 50002"        "50002" "$(req -X PUT "$BASE/sessions/$SESS_ID/supervisor-evaluation" "${AUTH_S[@]}" -H 'Content-Type: application/json' -d '{"objective":5}')" "d['code']"
want "草稿列表接口返回 50002"          "50002" "$(req "$BASE/drafts" "${AUTH_S[@]}")" "d['code']"

EV2="$(req -X PUT "$BASE/sessions/$SESS_ID/supervisor-observation" "${AUTH_S[@]}" -H 'Content-Type: application/json' \
  -d '{"comment":"verify.sh 审核意见（重提）","highlights":"亮点A2"}')"
want "观察幂等覆盖 comment 生效"       "verify.sh 审核意见（重提）" "$EV2" "d['data']['comment']"
want "幂等覆盖后评价仍为 1 条"         "1"   "$(req "$BASE/sessions/$SESS_ID/evaluation" "${AUTH_S[@]}")" "len(d['data']['supervisorScores'])"
want "新增后课程 1 授课记录总数=4"     "4"   "$(req "$BASE/courses/1/sessions" "${AUTH_S[@]}")" "d['data']['total']"

echo
echo "${c_dim}== 13.0 常用语标签库（N2）==${c_reset}"
want "常用语标签库可用"                "0"     "$(req "$BASE/supervision/comment-tags" "${AUTH_S[@]}")" "d['code']"
want "标签分组非空"                    "True"  "$(req "$BASE/supervision/comment-tags" "${AUTH_S[@]}")" "len(d['data'])>0"
want "教师访问标签库 40301"            "40301" "$(req "$BASE/supervision/comment-tags" "${AUTH_T[@]}")" "d['code']"

echo
echo "${c_dim}== 13.1 AI 评析任务链（D3/R0.2，无 Key 时熔断且页面可用）==${c_reset}"
AN="$(req -X POST "$BASE/sessions/$SESS_ID/ai-observation/analyze" "${AUTH_S[@]}")"
want "督导入队 AI 评析成功"           "0"     "$AN" "d['code']"
AI_TASK_ID="$(json "$AN" "d['data']['taskId']")"
want "AI 观察详情可读（页面不被阻断）" "0"     "$(req "$BASE/sessions/1/ai-observation" "${AUTH_S[@]}")" "d['code']"
want "AI 任务状态查询可用"            "0"     "$(req "$BASE/ai-tasks/$AI_TASK_ID" "${AUTH_S[@]}")" "d['code']"
want "教师查他人 AI 观察 40302"       "40302" "$(req "$BASE/sessions/4/ai-observation" "${AUTH_T[@]}")" "d['code']"
want "教师重试 AI 任务 40301"         "40301" "$(req -X POST "$BASE/ai-tasks/$AI_TASK_ID/retry" "${AUTH_T[@]}")" "d['code']"
want "AI 任务不存在 40401"            "40401" "$(req "$BASE/ai-tasks/999999" "${AUTH_S[@]}")" "d['code']"

echo
echo "${c_dim}== 13.2 教师建课/录音覆盖/删除/本人回听（B1/B3/B4/D6/R1.1）==${c_reset}"
TSESS="$(req -X POST "$BASE/sessions" "${AUTH_T[@]}" -H 'Content-Type: application/json' \
  -d '{"courseId":1,"classId":2,"sessionDate":"2026-09-11","period":"1-2 节","topic":"教师自建"}')"
want "教师为本人课程建课成功"           "0"     "$TSESS" "d['code']"
TSESS_ID="$(json "$TSESS" "d['data']['id']")"
want "教师为他人课程建课 40302"         "40302" "$(req -X POST "$BASE/sessions" "${AUTH_T[@]}" -H 'Content-Type: application/json' -d '{"courseId":2,"classId":3,"sessionDate":"2026-09-11","period":"1-2 节"}')" "d['code']"

TMPAUDIO="$(mktemp /tmp/aijiaoxue-verify-XXXXXX.mp3)"
printf 'ID3\003\000\000\000\000\000\000\000\000\000\000' > "$TMPAUDIO"
RP="$(req -X POST "$BASE/sessions/$TSESS_ID/recording" "${AUTH_T[@]}" -F "file=@$TMPAUDIO")"
want "教师上传录音成功"                 "0"     "$RP" "d['code']"
RP2="$(req -X POST "$BASE/sessions/$TSESS_ID/recording" "${AUTH_T[@]}" -F "file=@$TMPAUDIO")"
want "重复上传为覆盖语义（非 409）"      "0"     "$RP2" "d['code']"
want "上传者本人可回听（playbackUrl 非空）" "True" "$(req "$BASE/sessions/$TSESS_ID/transcript" "${AUTH_T[@]}")" "len(d['data']['recording']['playbackUrl'])>0"
want "教师删除本人录音成功"             "0"     "$(req -X DELETE "$BASE/sessions/$TSESS_ID/recording" "${AUTH_T[@]}")" "d['code']"
want "教师删除本人授课记录成功"         "0"     "$(req -X DELETE "$BASE/sessions/$TSESS_ID" "${AUTH_T[@]}")" "d['code']"
want "删除后场次不存在 40401"           "40401" "$(req "$BASE/sessions/$TSESS_ID" "${AUTH_T[@]}")" "d['code']"
rm -f "$TMPAUDIO"

echo
echo "${c_dim}== 13.3 最小课表下拉与建课班级必填（B11）==${c_reset}"
TT="$(req "$BASE/timetable/mine" "${AUTH_T[@]}")"
want "教师课表可读且非空"                "True"  "$TT" "len(d['data'])>0"
want "教师课表只含本人条目"              "True"  "$TT" "len([x for x in d['data'] if x['teacherId']!=2])==0"
want "教师传他人 id 仍只返回本人课表"    "True"  "$(req "$BASE/timetable/mine?teacherId=3" "${AUTH_T[@]}")" "len([x for x in d['data'] if x['teacherId']!=2])==0"
want "建课缺班级返回 40002"              "40002" "$(req -X POST "$BASE/sessions" "${AUTH_T[@]}" -H 'Content-Type: application/json' -d '{"courseId":1,"sessionDate":"2026-09-11","period":"1-2 节"}')" "d['code']"
want "建课班级错配返回 40002"            "40002" "$(req -X POST "$BASE/sessions" "${AUTH_T[@]}" -H 'Content-Type: application/json' -d '{"courseId":1,"classId":3,"sessionDate":"2026-09-11","period":"1-2 节"}')" "d['code']"
want "教师维护课表返回 40301"            "40301" "$(req -X POST "$BASE/timetable" "${AUTH_T[@]}" -H 'Content-Type: application/json' -d '{"courseId":1,"classId":1}')" "d['code']"
NT="$(req -X POST "$BASE/timetable" "${AUTH_S[@]}" -H 'Content-Type: application/json' -d '{"courseId":8,"classId":14}')"
want "督导新增课表成功"                  "0"     "$NT" "d['code']"
NT_ID="$(json "$NT" "d['data']['id']")"
want "重复课表返回 40901"                "40901" "$(req -X POST "$BASE/timetable" "${AUTH_S[@]}" -H 'Content-Type: application/json' -d '{"courseId":1,"classId":1,"teacherId":2}')" "d['code']"
want "督导删除课表成功"                  "0"     "$(req -X DELETE "$BASE/timetable/$NT_ID" "${AUTH_S[@]}")" "d['code']"

echo
echo "${c_dim}== 14. 提优闭环：清单/反思/站内信（N4/N5/B12）==${c_reset}"
IM="$(req -X POST "$BASE/improvement-items" "${AUTH_T[@]}" -H 'Content-Type: application/json' \
  -d '{"title":"增加课堂互动","sessionId":1,"source":"self"}')"
want "教师创建改进项成功"               "0"         "$IM" "d['code']"
IM_ID="$(json "$IM" "d['data']['id']")"
want "改进项初始状态=open"              "open"      "$IM" "d['data']['status']"
want "教师自评完成=closed（2026-10-20 简化）" "closed" "$(req -X PUT "$BASE/improvement-items/$IM_ID/complete" "${AUTH_T[@]}" -H 'Content-Type: application/json' -d '{"doneNote":"已改进"}')" "d['data']['status']"
want "主任访问清单返回 40301"           "40301"     "$(req "$BASE/improvement-items" "${AUTH_D[@]}")" "d['code']"
want "教师创建反思成功"                 "0"         "$(req -X POST "$BASE/teaching-reflections" "${AUTH_T[@]}" -H 'Content-Type: application/json' -d '{"sessionId":1,"content":"反思内容"}')" "d['code']"
want "教师反思列表非空"                 "True"      "$(req "$BASE/teaching-reflections" "${AUTH_T[@]}")" "len(d['data'])>0"
want "督导提交观察成功（站内信前置）"   "0"         "$(req -X PUT "$BASE/sessions/1/supervisor-observation" "${AUTH_S[@]}" -H 'Content-Type: application/json' -d '{"comment":"站内信测试"}')" "d['code']"
DEC="$(req "$BASE/sessions/1/evaluation" "${AUTH_S[@]}")"
want "P7.4 督导观察后 supervisorEvaluated=true" "True"     "$DEC" "d['data']['supervisorEvaluated']"
want "P7.4 督导观察不改 AI 场次状态"            "evaluated" "$DEC" "d['data']['session']['status']"
want "P7.4 计划 1 的 evaluated 仍为 true"       "True"     "$(req "$BASE/supervision/plans?page=1&pageSize=10" "${AUTH_S[@]}")" "[p['evaluated'] for p in d['data']['list'] if p['id']==1][0]"
want "教师收到站内信"                   "True"      "$(req "$BASE/notifications" "${AUTH_T[@]}")" "d['data']['total']>0"
want "站内信全部已读成功"               "0"         "$(req -X PUT "$BASE/notifications/read-all" "${AUTH_T[@]}")" "d['code']"
want "全部已读后未读数=0"               "0"         "$(req "$BASE/notifications" "${AUTH_T[@]}")" "d['data']['unread']"

# 清理本轮提优闭环新增数据，保持种子纯净
mysql -h 127.0.0.1 -P "${MYSQL_PORT:-3306}" -u aijiaoxue -paijiaoxue_dev aijiaoxue \
  -e "DELETE FROM improvement_items; DELETE FROM teaching_reflections; DELETE FROM user_notifications; DELETE FROM ai_tasks; DELETE FROM teacher_ai_reports WHERE teacher_id=2 AND semester='2026-2027-1' AND course_id=0;" >/dev/null 2>&1 \
  && echo "${c_dim}已清理提优闭环与 P3 smoke 新增数据${c_reset}"

echo
echo "${c_dim}== 15. 报告导出（N9）与演示模式（B6/N7）==${c_reset}"
REP="$(req "$BASE/teachers/2/report" "${AUTH_D[@]}")"
want "教师报告接口可用"                 "0"     "$REP" "d['code']"
want "报告综合分=72.5"                  "72.5"  "$REP" "d['data']['summary']['compositeScore']"
want "报告含最近督导评语"               "True"  "$REP" "len(d['data']['recentComments'])>0"
want "报告含待改进清单字段"             "True"  "$REP" "'improvementItems' in d['data']"
want "报告含教师姓名=李明"              "李明"  "$REP" "d['data']['teacherName']"
want "教师查他人报告 40302"             "40302" "$(req "$BASE/teachers/3/report" "${AUTH_T[@]}")" "d['code']"

DEMO="$(req -X POST "$BASE/demo/seed" "${AUTH_S[@]}")"
DEMO_CODE="$(json "$DEMO" "d['code']")"
if [ "$DEMO_CODE" = "0" ]; then
  want "演示数据载入成功"               "0"  "$DEMO" "d['code']"
  want "演示转写条数=14"                "14" "$DEMO" "d['data']['transcripts']"
  want "演示载入后可读示例转写"          "done" "$(req "$BASE/sessions/1/transcript" "${AUTH_S[@]}")" "d['data']['transcript']['status']"
else
  want "演示模式未开启返回 40301"       "40301" "$DEMO" "d['code']"
fi

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
