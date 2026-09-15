# 贡献指南 —— 「爱教学」协作与提交流程

> **适用对象**：所有参与本仓库开发的成员（含 AI 编码助手）。
> **核心原则**：**任何改动都不得直接推到 `main`**，必须走「建分支 → 自检 → 提 PR → 评审 → 合入」。
> 本文只讲**怎么做**；技术规范见 `docs/` 下的编码宪法。

---

## 1. 开工前的阅读顺序

按顺序读，不要跳：

| 顺序 | 文档 | 重点章节 |
|:----:|------|---------|
| 1 | `README.md` | 「For Developers」——环境准备、启停命令、验证方式 |
| 2 | 你负责的那一端 | `docs/backend_AGENTS.md` 或 `docs/frontend_AGENTS.md` |
| 3 | `docs/MySQL数据库创建指导.md` | 涉及数据、表结构、SQL 口径时必读 |
| 4 | `docs/Sprint2-3-教学评价与提优-开发计划.md` | 当前迭代的功能与规范 |
| 5 | `docs/Sprint2-3-任务清单.md` | 找到你被分配的任务编号 |

**两端各自最不能漏的章节**：

- 后端：§3 角色与数据范围规则、§5 目录结构、§8 接口契约、§10 分层与数据流
- 前端：§5 目录结构、§9 美术与设计规范、§10 数据模型与接口约定

---

## 2. 一次性准备

```bash
# ① 克隆（走 SSH，需先配置 GitHub SSH key）
git clone git@github.com:kaix-rbq/I-teaching-Web-app.git
cd I-teaching-Web-app

# ② 确认提交身份（提交记录里显示的名字）
git config user.name  "你的名字"
git config user.email "你的邮箱"

# ③ 按 README「For Developers」完成环境准备与数据库初始化
#    国内网络务必先配置 GOPROXY，否则首次 make run 会超时
go env -w GOPROXY=https://goproxy.cn,direct
```

---

## 3. 分支规范

| 项 | 规范 |
|----|------|
| 命名 | `feature/<故事编号>-<短描述>`，如 `feature/S6.3-session-evaluation` |
| 修复 | `fix/<短描述>`，如 `fix/login-token-redirect` |
| 文档 | `docs/<短描述>` |
| 起点 | **必须从最新 `main` 切出** |
| 粒度 | 一个分支只做一件事；不要把无关改动混在同一个 PR |
| 生命周期 | 合入后立即删除，不要长期保留 |

---

## 4. 标准动作序列（复制即用）

```bash
# ① 同步 main —— 每次开工前必做，避免基于旧代码开发
git switch main
git pull --ff-only origin main

# ② 建分支
git switch -c feature/S6.3-session-evaluation

# ③ 开发
#    契约先行：改接口/表结构/设计约定，必须"先改文档，再改代码"（见 §5）

# ④ 自检 —— 必须全绿才能提交
cd backend  && make lint && make test && make build
cd ../frontend && npm run lint && npm run typecheck && npm run test && npm run build
#    涉及接口或数据口径时，再跑一次端到端验收：
#    cd backend && BASE=http://127.0.0.1:8080/api/v1 bash scripts/verify.sh

# ⑤ 提交 —— 先看清改了什么，再提交
git status
git diff --staged
git add <具体文件>              # 不要用 git add -A / git add .
git commit -m "feat(session): 新增授课记录列表接口"

# ⑥ 开发期间 main 有更新时，用 rebase 跟进（保持线性历史）
git fetch origin
git rebase origin/main

# ⑦ 推送
git push -u origin feature/S6.3-session-evaluation

# ⑧ 在 GitHub 开 Pull Request，目标分支选 main，粘贴 §10 的 PR 模板

# ⑨ 按评审意见继续修改：往同一分支继续 commit + push，PR 会自动更新
#    不要为同一件事开第二个 PR

# ⑩ 评审通过后合入（由评审人操作），然后清理分支
git switch main && git pull --ff-only origin main
git branch -d feature/S6.3-session-evaluation
git push origin --delete feature/S6.3-session-evaluation
```

---

## 5. 契约先行（本团队最容易出错的环节）

**规则：凡是改了"别人要跟着改"的东西，必须先改文档，再改代码。**

| 你改了什么 | 必须先改 | 再改 |
|-----------|---------|------|
| 接口路径 / 请求响应字段 | `docs/backend_AGENTS.md` §8 或开发计划 §4 | `internal/dto/` → `handler/` → `frontend/src/types/` → `frontend/src/api/` |
| 表结构 / 字段 | `docs/MySQL数据库创建指导.md` 或开发计划 §3 | `backend/database/schema.sql` → `internal/model/` → `repository/` |
| 评分维度 / 权重 / 聚合口径 | 开发计划 §2 | `pkg/scoring` → `service/` |
| 权限 / 数据范围 | 开发计划 §5 | `middleware/` → `service/` |
| 路由 / 页面跳转 | `docs/frontend_AGENTS.md` §6 或开发计划 §6 | `router/index.ts` → `views/` |
| 关键设计约定 | 开发计划 §10 | 对应实现 |

> **顺序反了就是返工**：先写代码后补文档，PR 评审时会被打回，且前端可能已经在错误的契约上开发完了。

---

## 6. 红线清单（绝对禁止）

| # | 禁止 | 后果 | 正确做法 |
|:-:|------|------|---------|
| 1 | 直接 `git push` 到 `main` | 绕过评审，覆盖团队代码 | 开分支提 PR |
| 2 | `git push --force` / `-f` | **重写已共享历史，抹掉别人的提交** | 撤销已推送的提交用 `git revert`；只有自己的 feature 分支且已 rebase 时，才可用 `git push --force-with-lease` |
| 3 | `git reset --hard` 后强推 | 同上，且本地未提交改动永久丢失 | 用 `git revert <sha>` 生成反向提交 |
| 4 | `git add -A` / `git add .` 后不看 diff 就提交 | 会把 `config.yaml`、`.devtools/`、临时文件带进仓库 | `git status` + `git diff --staged` 确认后再 `git add <具体文件>` |
| 5 | 提交 `backend/config.yaml`、`.env*`、`uploads/`、`.devtools/` | 泄露密钥、隐私数据、本地环境 | 这些已被 `.gitignore` 覆盖；若已误提交立即告知成员四 |
| 6 | 在别人的分支上直接 push | 覆盖对方未完成的工作 | 在 PR 里评论建议，或协商后由本人修改 |
| 7 | `git checkout .` / `git clean -fd` 清理不认识的改动 | 可能是同事未提交的工作 | 先 `git stash` 保存，再操作 |
| 8 | 对共享数据库执行 `make db-seed` | **`seed.sql` 会 TRUNCATE 全部 6 张表，数据清空** | 只对本地库执行 |
| 9 | 自己 approve 自己的 PR 并合入 | 评审形同虚设 | 至少 1 名成员 approve 后才能合入 |
| 10 | 在 `main` 上直接改文件后 `git commit` | 落在 main 上，无法评审 | 发现后立即 `git switch -c <分支>` 把提交带走，再 `git switch main && git reset --hard origin/main` |

---

## 7. 冲突解决

```bash
git fetch origin
git rebase origin/main
# 若出现冲突：
git status                       # 查看冲突文件
#   → 手工编辑，删掉 <<<<<<< / ======= / >>>>>>> 标记，保留正确内容
git add <冲突文件>
git rebase --continue

# 想放弃这次 rebase：
git rebase --abort
```

> ⚠️ **冲突解决后必须重跑 §4 的自检**——冲突解决很容易把别人的改动或自己的改动改坏。
> ⚠️ rebase 后推送自己分支用 `git push --force-with-lease`（**绝不加 `-f` 到 main**）。

---

## 8. 事故补救

| 事故 | 处置 |
|------|------|
| 不小心把提交推到 `main` 了 | **不要** rebase main。立即开分支做 `git revert <sha>`，提 PR 合入；并在群里说明 |
| 误提交了密钥 / `config.yaml` | 立即通知成员四。**仅 revert 不够**（历史里仍有），密钥必须作废重发 |
| 本地改动不见了 / 被覆盖 | `git reflog` 找到丢失的 commit → `git switch -c rescue <sha>` 救回 |
| 推错了分支 | `git branch -m <正确名>` 本地改名；远端 `git push origin --delete <错分支>` |
| 提交信息写错但还没 push | `git commit --amend` |
| 提交信息写错且已 push | **不要改历史**，在 PR 描述里更正即可 |
| `git pull` 报 non-fast-forward | 说明你分支落后或有本地提交，改用 `git fetch && git rebase origin/main` |
| 不确定当前状态 | 先 `git status` + `git log --oneline -5`，**看不懂就不要继续敲命令**，问成员四 |

---

## 9. 评审要求

**谁审**（摘自两份 AGENTS）：

- 后端改动：成员四（胡凯翔）初审 → 成员一 / 成员三复审
- 前端改动：成员二（刘子杰）初审 → 成员一 / 成员三复审
- 涉及表结构、接口契约、权限的改动：**必须经成员四确认**

**看什么**（评审人按此 5 条检查）：

1. **分层是否越界** —— handler 有没有直连 repository、有没有在 handler 写业务、`api/` 外有没有 import axios
2. **契约与文档是否同步** —— §5 表格里对应的文档是否已更新
3. **数据范围是否在后端裁剪** —— 绝不能依赖前端过滤实现权限
4. **越权与隐私风险** —— 教师能否看到他人评分/评语、音频是否越界可见
5. **自检是否通过** —— PR 模板里的 checklist 是否逐项勾选

**合入规则**：至少 1 人 approve；**作者不得自审自合**。

---

## 10. PR 描述模板（复制到 PR 正文）

```markdown
## 变更内容
<!-- 一句话说明做了什么、为什么 -->

## 关联
- 故事编号：S6.x / T1.x
- 设计文档：docs/Sprint2-3-教学评价与提优-开发计划.md §x

## 自检清单（未过不要提 PR）
- [ ] 后端改动：`make lint && make test && make build` 全绿
- [ ] 前端改动：`npm run lint && npm run typecheck && npm run test && npm run build` 全绿
- [ ] 涉及接口/数据口径：`bash scripts/verify.sh` 全绿
- [ ] **契约先行**：接口字段变更已先更新 `docs/backend_AGENTS.md` §8 或开发计划 §4
- [ ] **表结构变更**：已更新 `docs/MySQL数据库创建指导.md` / 开发计划 §3 + `database/schema.sql` + `internal/model/`
- [ ] **前端类型同步**：`frontend/src/types/` 已随契约更新
- [ ] **设计约定变更**：已更新开发计划 §10 关键设计约定速查
- [ ] 数据范围裁剪写在 service 层，未依赖前端过滤
- [ ] 未提交 `config.yaml` / `.env*` / `uploads/` / `.devtools/`
- [ ] 分支已 rebase 到最新 `main`

## 影响面
- [ ] 涉及表结构变更（需数据库迁移）
- [ ] 涉及接口契约变更（需通知前端）
- [ ] 涉及权限 / 数据范围
- [ ] 仅文档 / 注释 / 格式

## 给评审人的说明
<!-- 需要重点看的地方、已知取舍、未覆盖的场景 -->
```

---

## 11. 卡住了找谁

| 情况 | 找谁 |
|------|------|
| 环境搭建、启动失败、数据库连不上 | 成员四（后端架构） |
| 前端页面、组件、样式规范 | 成员二（前端） |
| 接口契约争议、需求范围、优先级 | 成员一（DRI / 产品） |
| 表结构变更审批 | 成员四 |
| **Git / 分支 / 合并操作不确定** | **先别继续敲命令**，截图当前 `git status` 问成员四 |

> 「不确定就先停手」是本文最重要的一条。Git 的大部分事故都来自"以为没事"的连续操作。

---

*维护：随协作方式变更同步更新 · 参考 `docs/backend_AGENTS.md` §16、`docs/frontend_AGENTS.md` §14*
