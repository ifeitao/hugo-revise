# hugo-revise

语言：中文 | English version: [README.md](README.md)

![版本](https://img.shields.io/github/v/release/your-username/hugo-revise)
![Go 版本](https://img.shields.io/badge/go-1.22%2B-00ADD8?logo=go)
![构建状态](https://img.shields.io/github/actions/workflow/status/your-username/hugo-revise/ci.yml?label=build)

最小可用的 Hugo 内容修订 CLI 工具，支持版本历史管理。

## 特性

- ✅ **重大修订跟踪**：专为内容重大修订或重写设计，不是 Git 的替代品
- ✅ 支持单文件（`.md`）和页面捆绑包（`index.md`）两种形式
- ✅ 使用 `.revisions` 独立目录存储历史版本，避免 Hugo 嵌套 bundle 限制
- ✅ 通过 `hugo list all` 准确获取页面 URL，完美支持 permalink 配置
- ✅ 基于日期的版本管理（每天最多一个修订版本）
- ✅ 归档版本不出现在列表中但可直接访问（`build.list: never, render: true`）
- ✅ 简单的 undo 功能撤销最后一次修订

## 安装

### 从 GitHub 安装（推荐）

```sh
go install github.com/your-username/hugo-revise/cmd/hugo-revise@latest
```

### 从源码安装

```sh
# 克隆仓库
git clone https://github.com/your-username/hugo-revise.git
cd hugo-revise

# 安装到 $GOPATH/bin
go install ./cmd/hugo-revise
```

## 使用

### 基本用法

```sh
# 对单个 .md 文件创建修订
hugo-revise content/posts/my-post.md

# 或者不带 .md 扩展名（自动检测）
hugo-revise content/posts/my-post

# 对页面捆绑包创建修订
hugo-revise content/posts/my-bundle

# 显式使用 revise 子命令
hugo-revise revise content/posts/my-post
```

### 撤销操作

```sh
# 撤销上一次修订
hugo-revise undo
```

### 目录结构

**单文件场景**：
```
content/posts/
├── my-post.md                      # 当前版本
└── my-post.revisions/              # 修订历史目录
    ├── 2025-11-30.md              # 11月30日归档
    └── 2025-12-01.md              # 12月1日归档
```

**页面捆绑包场景**：
```
content/posts/
├── my-bundle/                      # 当前版本捆绑包
│   ├── index.md
│   └── image.png
└── my-bundle.revisions/            # 修订历史目录
    ├── 2025-11-30/
    │   ├── index.md
    │   └── image.png
    └── 2025-12-01/
        └── index.md
```

### 生成的 URL

归档版本的 URL 自动添加 `/revisions/` 路径段：

- 当前页面：`/my-post/`
- 归档（11月30日）：`/my-post/revisions/2025-11-30/`
- 归档（12月1日）：`/my-post/revisions/2025-12-01/`

## 配置 `.hugo-reviserc.toml`

在 Hugo 项目根目录创建配置文件以自定义日期格式：

```toml
[versioning]
date_format = "2006-01-02"  # 默认格式，可根据需要自定义
```

**注意**：本工具仅支持基于日期的版本管理。日期格式遵循 Go 语言的时间格式化约定。

## Front Matter 字段

### 当前版本

```yaml
---
title: 我的文章
date: 2025-12-01T00:10:08+08:00              # 更新为当前时间（代表修订日期）
lastmod: 2025-12-01T00:10:08+08:00           # 最后修改时间（自动更新为当前时间）
revisions_history:                           # 所有版本列表（按时间排序）
  - 2024-06-15
  - 2025-12-01
---
```

### 归档版本

```yaml
---
title: 我的文章
date: 2024-06-15                             # 原始日期（保持不变）
lastmod: 2024-06-15T10:30:00+08:00           # 原始 lastmod（保持不变）
url: "/my-post/revisions/2024-06-15/"       # 固定 URL（归档版本专用）
build:
  list: never                                # 不出现在列表中
  render: true                               # 但可以被直接访问
revisions_history:                           # 与当前版本相同的版本列表
  - 2024-06-15
  - 2025-12-01
---
```

## URL 推导优先级

1. **现有 url 字段**：如果 front matter 中已有 `url` 字段，直接使用
2. **Hugo list all**：运行 `hugo list all` 获取实际 URL（尊重 permalink 配置）
3. **slug 字段**：结合 section 和 slug 生成（如 `slug: my-post` → `/posts/my-post/`）
4. **路径推导**：根据文件路径推导（如 `content/posts/demo` → `/posts/demo/`）

## Hugo 集成

### 显示修订历史

可选：将 `templates/layouts/partials/revision-history.html` 复制到你的 Hugo 项目：

```sh
mkdir -p your-hugo-project/layouts/partials
cp templates/layouts/partials/revision-history.html \
   your-hugo-project/layouts/partials/
```

在文章模板中引用：

```go-html-template
{{ partial "revision-history.html" . }}
```

## 注意事项

- **工具定位**：hugo-revise 用于跟踪内容的重大修订（重写、显著更新），不用于日常编辑。请使用 Git 进行粒度版本控制。
- **每天一个修订**：如果尝试在同一天创建多个修订，会收到错误提示。这是有意设计的。
- **建议在修订前提交工作树**：`git add -A && git commit -m "before revision"`
- **需要 Hugo 可执行文件**：确保 `hugo` 命令在 PATH 中可用
- **在 Hugo 项目根目录运行**：工具需要找到 `hugo.toml` 等配置文件
- **Hugo 0.145+**：使用 `build` 字段而非已废弃的 `_build`
- **Front Matter 字段行为**：
  - `date`：当前版本更新为当前时间（代表修订日期）；归档版本保留原始值
  - `lastmod`：当前版本更新为当前时间；归档版本保留原始值
  - `revisions_history`：当前版本和归档版本都会添加，包含所有版本日期的按时间排序列表
  - `url`：仅添加到归档版本，确保固定的永久链接
  - `build`：仅添加到归档版本，防止在列表页面中显示
  - 版本标签基于修订日期（每天最多一个修订版本）

## 开发计划

### M1 - 基础功能（已完成）
- ✅ 单文件和捆绑包支持
- ✅ `.revisions` 独立目录结构
- ✅ 基于 `hugo list all` 的 URL 检测
- ✅ 版本冲突处理
- ✅ 基础 undo 功能

### M2 - 增强功能（计划中）
- 支持自定义版本标签（如 v1、v2、v3）
- 更完善的模板示例，提供更好的样式和功能
