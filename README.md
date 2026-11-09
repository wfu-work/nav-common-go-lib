# nav-common-go-lib

`nav-common-go-lib` 是一个面向 `nav-app` 场景的 Go 后端基础库，基于 `Gin`、`Gorm`、`Viper`、`Zap`、`JWT`、`Casbin` 等组件封装了一套可复用的启动流程、公共中间件、基础服务与默认系统模块。

它更适合作为业务服务的底座库使用，而不是一个独立的完整应用。业务项目接入后，可以快速拥有统一的配置管理、鉴权、路由组织、数据库初始化、日志、缓存和定时任务能力。

## 特性

- 基于 `Gin` 的统一 Web 初始化与路由挂载
- 基于 `Gorm` 的数据库接入与表自动迁移
- 支持 `SQLite`、`MySQL`、`PostgreSQL`
- 支持 `Redis`、本地缓存、日志持久化
- 内置 `JWT` 鉴权、`Casbin` 权限控制、`CORS`
- 内置用户、角色、企业、登录、文件、系统配置等基础模块
- 支持配置热更新、优雅关闭、定时任务扩展
- 提供 `SysInit` 钩子，便于业务层无侵入扩展

## 安装

```bash
go get github.com/wfu-work/nav-common-go-lib
```

当前模块路径：

```go
module github.com/wfu-work/nav-common-go-lib
```

## 环境要求

- Go `1.27.3+`（与 `go.mod` 中的最低版本一致）
- SQLite 使用官方 Gorm 驱动，编译时需要启用 CGO 并提供 C 编译器
- 推荐在独立业务项目中作为基础库引用
- 如果启用对应能力，需要准备：
  - `SQLite` / `MySQL` / `PostgreSQL`
  - `Redis`

## 适用场景

- 内部管理后台
- 标准 Go Web 后端
- 账号、角色、权限、登录等通用基础模块
- 多个业务项目共享统一后端底座的场景

如果你的项目只需要零散工具函数，或者完全不需要 HTTP、数据库、鉴权、配置初始化能力，这个库可能会偏重。

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

## 快速开始

下面是一个最小接入示例：

```go
package main

import (
	"github.com/gin-gonic/gin"
	"github.com/wfu-work/nav-common-go-lib/inits"
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
		// HTTP 请求与后台任务结束后，释放业务资源
	})

	if err := sys.InitE(); err != nil {
		panic(err)
	}
}
```

默认启动过程中会完成：

- 加载配置文件并监听热更新
- 初始化日志
- 初始化 Redis、本地缓存、IP 地理库
- 初始化数据库连接与默认表
- 注册系统路由
- 启动 HTTP 服务并监听退出信号

## 配置说明

库内部带有一份基础配置 `config-common.yaml`，业务项目还需要提供自己的配置文件。

配置文件查找优先级：

1. 启动参数 `-c /path/to/config.yaml`
2. 环境变量 `NAV_CONFIG`
3. 根据 `gin.Mode()` 自动选择：
   - `config.debug.yaml`
   - `config.release.yaml`
   - `config.test.yaml`
4. 根目录默认配置 `config.yaml`

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
  trusted-proxies: []
  max-request-bytes: 16777216
  max-upload-bytes: 33554432
  enable-swagger: false

jwt:
  issuer: nav-app
  signing-key: replace-with-a-real-secret
  expires-time: 7d
  buffer-time: 1d
  cookie-secure: false # HTTPS 反向代理部署时设为 true
  cookie-same-site: lax

cors:
  mode: whitelist
  whitelist:
    - allow-origin: https://admin.example.com
      allow-credentials: true

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
  max-idle-conns: 1
  max-open-conns: 1

zap:
  level: info
  format: console
  director: logback
  log-in-console: true
```

运行时使用 `global.Config()` 读取只读配置快照；需要修改时先 `Clone()`，由 `global.SetConfig()` 发布。请求中通过 `global.RequestConfig(c)` 获取本次请求固定的快照。原 `global.NAV_CONFIG` 仅保留启动输入用途，运行期间不要直接读写。扩展配置使用 `global.RegisterExtra(key, value)` 注册，注册后的值应保持不可变。

文件热更新只接受 JWT 有效期与刷新窗口、CORS 和验证码设置。数据库、Redis、日志、路径、路由、密钥、Cookie、代理与请求上限等修改需要重启；校验失败或涉及重启的文件变更不会替换当前快照。配置写入接口通过临时文件和重命名原子保存文件。

默认 CORS 为白名单模式，需要配置前端 Origin 才能跨域读取响应。JWT Cookie 默认为 host-only、HttpOnly、SameSite=Lax；HTTPS 反向代理场景请设置 `cookie-secure: true`，同时显式配置可信代理。普通请求体上限为 16 MiB，上传请求为 32 MiB（包含 multipart/base64 编码开销），文件服务还会检查实际文件大小。

## 默认路由

默认内置一批系统级接口，主要包括：

- `/health` 健康检查
- `/swagger/index.html` Swagger 文档入口（显式启用后注册）
- 用户、角色、企业、登录、登录日志
- 文件上传与访问
- 系统配置
- 服务器信息
- 注册与加解密接口

这些路由会统一挂载到 `system.router-prefix` 下，例如当前前缀配置为 `/api` 时：

- `GET /api/health`
- `GET /api/swagger/index.html`

Swagger 默认关闭；启用 `system.enable-swagger` 后，由宿主应用导入自己生成的文档包。

## 扩展点

`inits.SysInit` 提供了一组钩子，方便业务项目扩展：

- `OnWebInit`：注册 Web 级初始化逻辑
- `OnRouterInit`：追加业务路由
- `OnTableInit`：初始化业务表
- `OnScheInit`：追加定时任务
- `OnCacheInit`：本地缓存初始化完成后执行
- `OnOtherInit`：追加其他初始化逻辑
- `OnClearInit`：注册数据库清理规则
- `OnShutInit`：请求与后台任务结束后、数据库等资源关闭前收尾

`InitContext(ctx)` 可由上层取消服务；`InitE()` 保留信号退出方式。关闭时先等待 HTTP 请求完成，再停止配置监听、定时任务，持久化缓存并排空审计队列，最后关闭 IP 库、Redis、数据库和日志。`Close()` 可重复调用；初始化失败也会回收已经创建的资源。

## 事务用法

基础 `CrudService` 已支持注入自定义 `*gorm.DB`，常用业务服务也提供了 `WithDB()`，适合在事务中复用：

```go
import (
	"context"

	"github.com/wfu-work/nav-common-go-lib/domains"
	"github.com/wfu-work/nav-common-go-lib/global"
	"github.com/wfu-work/nav-common-go-lib/services"
	"gorm.io/gorm"
)

func doBiz(ctx context.Context) error {
	return services.WithTx(ctx, func(tx *gorm.DB) error {
		userSvc := services.UserServiceApp.WithDB(tx)
		roleSvc := services.RoleServiceApp.WithDB(tx)
		companySvc := services.CompanyServiceApp.WithDB(tx)

		user, err := userSvc.GetByUserName("admin")
		if err != nil {
			return err
		}
		if user != nil {
			_, _, err = roleSvc.List(domains.PageInfo{Page: 1, Size: 10}, "name")
			return err
		}
		return companySvc.DeleteByGuid("some-guid")
	})
}
```

如果你已经手上有事务对象，也可以直接注入：

```go
tx := global.NAV_DB.Begin()
userSvc := services.UserServiceApp.WithDB(tx)
companySvc := services.CompanyServiceApp.WithDB(tx)
```

也可以使用 `services.NewServiceGroup(db)` 批量构造绑定数据库的业务服务，或在事务内传入 `tx`。HTTP 默认装配仍使用全局资源，完整应用应在独立进程运行。

当前已支持 `WithDB()` 的典型服务包括：

- `UserService`
- `RoleService`
- `CompanyService`
- `LoggerService`
- `LoginLogService`
- `UserRoleService`
- `RegisterService`

## 核心对象

初始化完成后，可通过 `global` 包访问公共资源：

- `global.NAV_DB`：Gorm 数据库连接
- `global.NAV_REDIS`：Redis 客户端
- `global.Config()`：系统配置只读快照
- `global.NAV_LOG`：Zap 日志对象
- `global.LOCAL_CACHE`：本地缓存

## 开发建议

- 依赖升级记录与优化分析见 [2026-11-05 检查报告](docs/dependency-upgrade-and-review-2026-11-05.md)
- 本次优化内容、行为变化与验证见 [代码优化实施说明](docs/optimization-implementation.md)
- 建议把这个库作为“基础设施层”使用，不要把所有业务逻辑都塞进公共库
- 业务实体、业务路由、业务服务建议在业务仓库中实现
- 生产环境必须显式配置 `jwt.signing-key`
- 建议区分 `debug`、`test`、`release` 三套配置文件
- 如果你需要给外部项目使用，优先保持模块路径与 GitHub 仓库一致

## 许可证

本项目使用 [MIT License](./LICENSE)。


## 企业与角色权限

平台身份只属于固定 GUID 的内置 `admin` 账号，所有权限及企业数据均可访问；`SUPER_ADMIN` 角色编码不能赋予平台身份。企业用户必须归属企业，各企业用户共用全局角色，角色管理与菜单授权仅由平台管理员维护，企业管理员可以为本企业用户分配全局角色。普通角色权限仅允许 `scope=company` 的权限点。菜单/按钮可通过 `dependencies` 声明依赖，其下 API 权限在保存授权时自动展开。

嵌入宿主在表结构及种子数据初始化完成、业务权限目录注册后，调用 `InitializeGlobalRoles` 完成角色升级与预置授权；旧 `InitializeCompanyRoles` 接口仍可调用，但只初始化全局角色。重复的启用内置角色会合并并转接用户关联，优先保留已配置的全局角色，否则采用最早已配置的旧角色，权限不会取并集；同编码自定义角色与停用角色保留原授权并调整冲突编码。用户企业归属继续生效；用户名包含已删除账号在内全局唯一，不允许复用。

`POST /company` 创建企业时，同时提交 `adminUsername` 和经 `/secret/encrypt` 加密的 `adminPassword`；企业与初始管理员在同一事务创建，初始管理员复用全局企业管理员角色。`POST /user` 支持 `roleGuidList`，也可通过 `PUT /user/:guid/roles` 单独分配。`GET /user/token` 返回 `company` 摘要、`abilityList`、`isPlatformAdmin`、`isCompanyAdmin` 和 `dataScope`。`PUT /company/current` 只修改当前企业允许编辑的资料。

每次请求重新检查用户、企业、当前角色和授权版本。保存账号、修改角色分配、修改密码或退出登录后，旧会话失效。内置 admin 与最后一位企业管理员受到保护。企业删除接口保留数据并拒绝删除，应使用停用功能。

认证白名单按 HTTP 方法和路径匹配。公共接口仅保留登录、验证码、加密和公开配置摘要；配置写入、解密和文件存储不再匿名开放。业务宿主仍必须给数据查询、修改及文件服务接入企业范围约束，不能仅依赖 Casbin 的 API 权限。

企业附件存入 `companies/<companyGuid>/`，静态读取及下载均校验实时身份和企业前缀。审计日志同时记录操作人企业与 `targetCompanyGuid`，便于追踪平台管理员对其他企业的操作。
