# AGENTS.md

本目录是「爱教学」后端仓库 `aijiaoxue-api`。

**编码前必读**：[`../docs/backend_AGENTS.md`](../docs/backend_AGENTS.md)（后端编码宪法：技术栈、目录结构、分层架构、接口契约、编码规则）。
**数据库指导**：[`../docs/MySQL数据库创建指导.md`](../docs/MySQL数据库创建指导.md)（建库/账号、表与迁移清单、种子与自查、GORM 对接、备份、排查）。
**开发操作规范**：[`../README.md`](../README.md) 的「For Developers」章节（启动/终止命令、扩展功能工作流、出错处置）。

关键纪律（摘自上述文档，务必遵守）：

- 依赖方向单向 `handler → service → repository → model`，禁止反向 import 与跨层跳调；
- 路由只在 `internal/router/router.go` 注册；`*gin.Context` 只在 `internal/handler/` 与 `internal/middleware/` 出现；
- **数据范围裁剪只发生在后端 service 层**，前端筛选参数不构成权限依据；
- 表结构事实源：Sprint 1 存量表看 `database/schema.sql`，Sprint 2 起新增/变更看 `migrations/`；**禁止 GORM `AutoMigrate`**；
- 接口契约变更必须**先改 `docs/backend_AGENTS.md` §8**，再改 `internal/dto/` 与前端 `src/types/`；
- 涉及 `docs/backend_AGENTS.md` §2.4 范围边界（Won't 清单）的需求，先亮红灯说明，不得直接实现。
