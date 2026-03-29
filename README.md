# nav-common-go-lib

`nav-common-go-lib` 是一个面向 `nav-app` 体系的 Go 后端公共库，基于 `Gin`、`Gorm`、`Viper`、`Zap`、`JWT`、`Casbin` 等组件封装了一套可复用的初始化流程、通用接口、基础中间件和公共服务。

这个库更适合作为业务服务的基础模块使用，而不是独立应用。业务项目引入后，可以快速获得以下能力：

- 服务初始化与优雅关闭
- 配置加载与热更新
- 多数据库接入能力：`MySQL`、`PostgreSQL`、`SQLite`
- Redis、日志、缓存等基础设施初始化
- JWT 鉴权、Casbin 权限控制、CORS 等中间件
- 用户、角色、企业、登录、文件、系统配置等通用 API / Service / Router
- 定时任务与数据库清理扩展点

## 项目结构

```text
.
├── apis/          HTTP 接口层
├── configs/       配置结构定义
├── dbs/           数据库初始化、连接与基础数据
├── domains/       领域模型 / 数据实体
├── global/        全局对象，如 DB、Redis、Logger、Config
├── inits/         系统启动与初始化入口
├── middlewares/   中间件与配置加载
├── response/      通用响应封装
├── routers/       路由注册
├── scheduleds/    定时任务
├── services/      业务服务层
├── utils/         工具方法
└── vos/           请求 / 响应 VO
```

## 环境要求

- Go `1.24+`
- 推荐准备一个业务项目作为启动入口
- 如需数据库能力，请准备以下之一：
  - `SQLite`
  - `MySQL`
  - `PostgreSQL`
- 如启用 Redis，请准备 Redis 实例

## 安装

在你的业务项目中引入：

```bash
go get navfirst.com/nav-common-go-lib
```

## 快速开始

下面是一个最小可运行示例：

```go
package main

import (
	"github.com/gin-gonic/gin"
	"navfirst.com/nav-common-go-lib/inits"
)

func main() {
	gin.SetMode(gin.DebugMode)

	sys := &inits.SysInit{}

	sys.OnWebInit(func(router *gin.Engine) {
		// 注册自定义全局中间件
	})

	sys.OnRouterInit(func(publicGroup *gin.RouterGroup, privateGroup *gin.RouterGroup) {
		// 注册业务路由
		publicGroup.GET("/ping", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "pong"})
		})
	})

	sys.OnTableInit(func() {
		// 注册业务表初始化逻辑
	})

	sys.OnOtherInit(func() {
		// 其他初始化逻辑
	})

	sys.OnShutInit(func() {
		// 优雅关闭前的资源释放
	})

	if err := sys.InitE(); err != nil {
		panic(err)
	}
}
```

启动后默认会完成：

- 加载配置文件
- 初始化日志
- 初始化缓存和 IP 地理库
- 初始化 Redis
- 初始化 Gorm 和基础表
- 注册系统默认路由
- 启动 HTTP 服务并监听退出信号

## 配置文件

库内部内置了一份基础配置 `config-common.yaml`，运行时还会继续读取外部业务配置文件。

配置文件查找优先级如下：

1. 启动参数 `-c /path/to/config.yaml`
2. 环境变量 `NAV_CONFIG`
3. 根据 `gin.Mode()` 自动选择：
   - `config.debug.yaml`
   - `config.release.yaml`
   - `config.test.yaml`
4. 回退到根目录下的 `config.yaml`

一个最小配置示例：

```yaml
system:
  app-name: nav-app
  addr: 8888
  db-type: sqlite
  router-prefix: /api
  oss-type: local
  use-redis: false
  use-strict-auth: false

jwt:
  issuer: nav-app
  signing-key: replace-with-your-secret
  expires-time: 7d
  buffer-time: 1d

local:
  oss-path: ./data/oss
  proxy-oss-path: /oss
  cache-path: ./data/cache.json
  ip2geo-path: ./data/ip2geo

sqlite:
  path: ./data/
  db-name: nav-app.db
  log-mode: error
  log-zap: false
  max-idle-conns: 10
  max-open-conns: 100

zap:
  level: info
  format: console
  director: logback
  log-in-console: true
```

## 默认路由

库中已经内置一批系统级路由，主要包括：

- `/health` 健康检查
- `/swagger/index.html` Swagger 文档入口
- 用户、角色、企业、登录、登录日志
- 文件上传与访问
- 系统配置
- 服务器信息
- 注册与加密相关接口

这些路由会统一挂载到 `system.router-prefix` 下，例如默认前缀为 `/api` 时：

- `GET /api/health`
- `GET /api/swagger/index.html`

## 扩展点

`inits.SysInit` 提供了一组钩子，便于业务项目在不修改公共库源码的情况下扩展：

- `OnWebInit`：注册全局 Web 初始化逻辑
- `OnRouterInit`：追加业务路由
- `OnTableInit`：初始化业务表
- `OnScheInit`：追加定时任务
- `OnCacheInit`：预留缓存初始化扩展
- `OnOtherInit`：其他业务初始化逻辑
- `OnClearInit`：注册数据库清理规则
- `OnShutInit`：服务关闭前收尾

## 核心全局对象

初始化完成后，可通过 `global` 包访问公共资源：

- `global.NAV_DB`：Gorm 数据库连接
- `global.NAV_REDIS`：Redis 客户端
- `global.NAV_CONFIG`：系统配置
- `global.NAV_LOG`：Zap 日志对象
- `global.LOCAL_CACHE`：本地缓存

## 适用场景

这个库适合以下类型的项目：

- 内部管理后台
- 通用账号 / 权限系统
- 基于 Gin 的标准后端服务
- 多个业务项目共享统一基础设施的场景

如果你的项目只需要纯工具函数，或者完全不需要 HTTP / 鉴权 / 数据库初始化，这个库可能会偏重。

## 开发建议

- 业务项目中建议只把这里作为“基础设施层”
- 业务实体、业务路由、业务服务建议放在各自项目中扩展
- 生产环境务必替换默认 JWT 密钥和数据库账号密码
- 建议明确区分 `debug`、`test`、`release` 三套配置文件

## 许可证

当前仓库内未看到明确的开源许可证文件。若需要对外分发或开源，建议补充 `LICENSE` 文件并明确使用范围。
