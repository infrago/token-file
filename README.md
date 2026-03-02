# token-file

`token-file` 是 `token` 模块的文件实现（driver: `github.com/infrago/token-file`）。

## 包定位

- 类型：驱动（Payloader + Revoker）
- 作用：轻量部署场景，用本地文件持久化 payload 与吊销信息

## 主要功能（v1）

- `Payloader`：按 tokenId 落盘 payload
- `Revoker`：落盘 token/tokenId 吊销状态
- TTL 到期后清理文件索引

## 专用配置（token.setting）

```toml
[token.setting]
file_path = "store/token.db"
```

## 状态

- 当前状态：`v1 设计稿`
- 计划：单机/边缘节点可选 store
