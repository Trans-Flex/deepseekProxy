# deepseek-proxy

一个用 Go 标准库实现的 DeepSeek 反向代理。鸿蒙 App 通过它调用大模型，**API key 只存在于代理层**，不暴露给客户端。

本项目由 [tinyProxyProject](https://github.com/Trans-Flex/tinyProxyProject) 演进而来——后者是用底层 socket 手写 HTTP 的正向代理，用于理解协议本身；本项目则切换到 `net/http`，聚焦业务代理能力。

## 请求流程

```
鸿蒙 App
│ POST /chat {"messages":[...]}
▼
deepseek-proxy (:8080)
│ 1. 读取请求体
│ 2. 注入 System Prompt（前插到 messages 头部）
│ 3. 强制覆盖 model 为 deepseek-flash
│ 4. 对最终请求体求 sha256，作为缓存 key
│ 5. 查 LRU 缓存 → 命中则直接返回
│ 6. 未命中 → singleflight 合并同 key 并发请求
│ 7. 构造 DeepSeek 请求，注入 Authorization
│ 8. 调用 api.deepseek.com，写回缓存
▼
DeepSeek API
```

## 快速开始

**1. 准备 API key**

在项目根目录创建 `.env`：DEEPSEEK_API_KEY=sk-你的key


`.env` 已在 `.gitignore` 中，不会进入版本库。

**2. 运行**

go run .
text


服务监听 `0.0.0.0:8080`。

**3. 测试**

```
curl.exe -X POST http://localhost:8080/chat ^
-H "Content-Type: application/json" ^
-d "{"messages":[{"role":"user","content":"帮我分析一下每天两小时学英语的计划"}]}"
```


## 接口契约

**请求**

```
POST /chat
Content-Type: application/json

{
    "messages": [
        {"role": "user", "content": "..."}
    ]
}
```

客户端只需发 `messages`。`model` 由代理强制指定，客户端传入会被忽略。

**响应**

原样透传 DeepSeek 的 OpenAI 兼容响应，客户端可直接读取 `choices[0].message.content`。

```
{
"choices": [
{
"message": {
"role": "assistant",
"content": "..."
}
}
],
"usage": { ... }
}
```

**错误**

上游调用失败时返回 `502`：

```
{"error": "upstream failed"}
```

## 设计决策

**为什么用 `net/http` 而不是手写 socket**

正向代理阶段手写 HTTP 是为了理解协议；反向代理场景下，`net/http` 已经处理好 HTTPS、连接池、keep-alive、Content-Length/chunked、header 规范化等细节，继续手写是重造轮子。本项目只保留 `cache.go` 中与 HTTP 无关的 LRU 核心。

**为什么用精确哈希缓存，而不是语义缓存**

精确哈希对自然语言问题的命中率天然有限（问法发散），它的实际收益来自两类同 key 场景：客户端网络重试、用户重复提交。语义缓存需要 embedding + 向量检索，成本和误判风险都不适合当前规模。

**singleflight 解决什么问题**

同一个 key 的并发请求（例如 50 个客户端同时问同一道题，或客户端重试风暴）会只触发一次上游调用，其余请求等待并共享结果。它不依赖缓存命中，只要并发同 key 就生效——这比缓存命中更常发生。

**API key 如何隔离**

- 通过 `.env` 加载到环境变量，不硬编码。
- 存入 handler 结构体，随请求生命周期存在。
- 只在 `deepseek.go` 构造请求头时使用一次。
- 不进响应体，不进日志输出。

客户端从头到尾不知道 key 的存在——这是引入代理层的根本原因。

**System Prompt 注入**

代理层强制在 `messages` 头部插入 system 消息，将模型行为限定为学习辅助。客户端无法绕过这一约束，也无需关心 prompt 的具体内容。

## 项目结构

```
.
├── main.go # 启动、依赖组装
├── handler.go # /chat 的 HTTP 处理：解析、注入、缓存调度、singleflight
├── deepseek.go # DeepSeek API 调用（唯一持有 key 的使用点）
├── cache.go # LRU 缓存（从 tinyProxyProject 复用）
├── .env # API key（不进版本库）
└── .gitignore
```

## 依赖

- `golang.org/x/sync/singleflight` — 并发请求合并
- `github.com/joho/godotenv` — 加载 `.env`

其余均为标准库。

