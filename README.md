# 我的宠物（My Pet）

家庭宠物档案管理应用：记录宠物档案、喂养信息、健康提醒和日常日志。

| 项目 | 内容 |
|------|------|
| 技术栈 | Go (Gin) + Vue3 (Vite, Pinia, Naive UI) + SQLite |
| 部署目标 | NAS（Docker / 直接运行） |
| 初始账号 | `admin` / `admin123`（首次启动自动创建，请部署后立即修改） |

## 功能

- **宠物档案**：创建/编辑/删除宠物，头像上传（限 2MB），删除时展示级联影响
- **喂养记录**：干粮/湿粮/零食 + 克数快速记录，按日期分组展示
- **健康提醒**：驱虫/疫苗/洗澡等周期提醒（每月/每季/每年/自定义），完成自动推进下个周期，月历视图标注提醒日期
- **日志**：行为/健康/其他分类的时间线日志
- **权限分级**：管理员全部操作；普通成员可查看全部、可添加喂养/日志/提醒/标记完成，不可删除任何数据、不可编辑宠物档案、不可管理用户
- **账号安全**：密码强度校验（8 位以上、含字母和数字）、连续 5 次登录失败锁定 15 分钟、JWT 24 小时有效
- **通知与备份**：提醒到期触发 Webhook（未配置则写入 `data/alerts.log`）、一键备份 SQLite 数据库

## 部署（Docker，推荐）

```bash
docker compose up -d -b
# 访问 http://<NAS-IP>:8080
```

数据持久化在 `./data`（SQLite 数据库、头像、备份、通知日志）。修改端口编辑 `docker-compose.yml` 中的映射即可。

生产环境建议在 `docker-compose.yml` 中把 `JWT_SECRET` 改为随机长字符串。

## 直接运行

依赖：Go 1.26+、Node 20+

```bash
cd frontend && npm install && npm run build && cd ..
go build -o my-pet .
PORT=8080 ./my-pet
```

> 注意：Go 纯 Go SQLite 驱动（modernc.org/sqlite）不支持 windows/386 工具链，Windows 32 位 Go 环境请用 `GOARCH=amd64 go build` 交叉编译。

## 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `PORT` | `8080` | 监听端口 |
| `JWT_SECRET` | `my-pet-secret-key-2026` | JWT 签名密钥，生产必须覆盖 |
| `DB_PATH` | `./data/my-pet.db` | SQLite 数据库路径 |

## 开发模式

```bash
# 终端 1：后端（默认 8080）
PORT=8080 go run .

# 终端 2：前端（5173，/api 代理到 8080；后端端口不同时设置 VITE_API_TARGET）
cd frontend && npm run dev
```

## API 概览

- 认证：`POST /api/auth/login`、`GET /api/auth/me`、`PUT /api/auth/password`
- 宠物：`GET/POST /api/pets`、`GET/PUT/DELETE /api/pets/:id`、`POST /api/pets/:id/avatar`
- 喂养：`POST/GET /api/pets/:id/feeds`、`DELETE /api/feeds/:id`
- 提醒：`POST/GET /api/pets/:id/reminders`、`PUT/DELETE /api/reminders/:id`、`POST /api/reminders/:id/complete`
- 日志：`POST/GET /api/pets/:id/logs`、`DELETE /api/logs/:id`
- 用户（管理员）：`GET/POST /api/users`、`DELETE /api/users/:id`
- 设置（管理员）：`GET/PUT /api/settings`、`POST /api/settings/webhook/test`、`GET /api/settings/notifications`、`POST /api/backup`
- 健康检查：`GET /api/ping`

## 自测

`scripts/smoke_test.sh` 覆盖认证、宠物/喂养/提醒/日志增删改查、权限、登录锁定、设置、备份、并发写入共 50+ 断言：

```bash
# 先启动服务（默认脚本指向 http://localhost:18080，可修改脚本中 BASE）
bash scripts/smoke_test.sh
```
