# Finance Invoice - PDF 发票批量识别系统

基于智谱 AI 多模态大模型的 PDF 发票批量识别 Web 服务。上传 PDF 发票，自动逐页识别，生成格式化 Excel。

## 功能

- 批量上传 PDF 发票，自动逐页识别
- 可控并发处理（默认 20 路并发）
- 任务完成后自动生成 Excel，支持下载
- 用户登录认证（CSV 文件配置）
- IP 白名单访问控制（支持 CIDR）
- 失败自动重试（指数退避，最多 10 次）
- 业务层异常重试（上传 + 识别全链路重试）

## Excel 输出格式

| 发票类型 | 发票号码 | 开票日期 | 购方名称 | 购方统一社会信用代码/纳税人识别号 | 销方名称 | 销方统一社会信用代码/纳税人识别号 | 项目名称 | 金额 | 税率 | 税额 | 价税合计（大写） | （小写） | 备注 | 开票人 | 原始文件名 | 识别状态 |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|

## 环境变量

| 变量 | 必填 | 默认值 | 说明 |
|------|------|--------|------|
| `ZHIPU_API_KEY` | 是 | - | 智谱 AI API Key |
| `USERS_CSV` | 否 | `users.csv` | 用户账号文件路径 |
| `ALLOWED_IPS` | 否 | 空（不限制） | IP 白名单，逗号分隔，支持 CIDR，如 `10.0.1.0/24,192.168.1.100` |
| `MAX_CONCURRENT` | 否 | `20` | 智谱 API 最大并发数 |
| `PORT` | 否 | `8080` | 服务端口 |
| `JWT_SECRET` | 否 | 内置默认值 | JWT 签名密钥 |

## 快速开始

### 直接运行

```bash
# 构建
go build -buildvcs=false -o finance-invoice .

# 启动
ZHIPU_API_KEY=your_api_key ./finance-invoice

# 带参数启动
ZHIPU_API_KEY=your_api_key \
  MAX_CONCURRENT=10 \
  ALLOWED_IPS=10.0.1.0/24 \
  PORT=9090 \
  ./finance-invoice
```

### Docker 运行

```bash
# 构建镜像
docker build -t finance-invoice .

# 启动容器
docker run -d \
  -p 8080:8080 \
  -e ZHIPU_API_KEY=your_api_key \
  -v $(pwd)/users.csv:/app/users.csv \
  -v $(pwd)/output:/app/output \
  finance-invoice
```

## 用户配置

CSV 文件格式（默认 `users.csv`）：

```csv
username,password
admin,admin123
```

## 项目结构

```
finance-invoice/
├── main.go                 # 入口，组装各模块
├── config/config.go        # 环境变量配置
├── auth/auth.go            # CSV 用户认证 + JWT
├── middleware/middleware.go # 认证 + IP 白名单中间件
├── zhipu/client.go         # 智谱 API 客户端（上传 + 识别）
├── task/task.go            # 任务管理（并发调度 + 重试 + 拆页）
├── handler/handler.go      # HTTP 接口层
├── excel/generate.go       # Excel 生成
├── web/index.html          # 前端页面
├── users.csv               # 用户账号
├── Dockerfile              # 容器构建
└── go.mod
```

## API 接口

| 方法 | 路径 | 认证 | 说明 |
|------|------|------|------|
| GET | `/` | 否 | 前端页面 |
| POST | `/api/login` | 否 | 登录，返回 JWT |
| POST | `/api/upload` | 是 | 上传 PDF 文件，创建识别任务 |
| GET | `/api/tasks` | 是 | 任务列表 |
| POST | `/api/tasks/clear` | 是 | 清理已结束的任务 |
| GET | `/api/tasks/:id` | 是 | 任务详情 |
| GET | `/api/tasks/:id/download` | 是 | 下载 Excel 结果 |

## 识别流程

1. 上传 PDF → `pdfcpu` 拆分为单页
2. 每页上传至智谱临时文件接口 → 获取公网 URL
3. 调用 `glm-5v-turbo` 多模态 API 识别发票字段
4. 解析 JSON 响应，提取 15 个发票字段
5. 全部完成后生成 Excel 供下载
6. 任一环节失败自动重试（指数退避，2s → 5min，最多 10 次）
