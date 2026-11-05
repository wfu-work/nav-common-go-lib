# 依赖升级与项目优化分析

检查日期：2026-11-05。检查对象：`github.com/wfu-work/nav-common-go-lib`。

本文保留依赖升级阶段的原始问题分析，位置和复现描述对应优化前代码。后续已完成代码优化，当前行为与验证结果见 [代码优化实施说明](optimization-implementation.md)。按项目要求，身份模块仅保留当前模型及正常初始化，不提供历史身份或索引兼容迁移。

## 本次升级结果

已更新 `go.mod`、`go.sum`，并同步 README 的 Go 版本要求。共升级 **56 个 Go 模块：7 个直接依赖、49 个间接依赖**；其余声明的依赖已经处于当前模块路径的最新版本。

| 直接依赖 | 原版本 | 升级版本 |
| --- | --- | --- |
| `github.com/casbin/casbin/v3` | v3.10.0 | v3.11.0 |
| `github.com/lionsoul2014/ip2region/binding/golang` | v0.0.0-20260709160242-cd40e3a1d532 | v0.0.0-20261028042204-6a276cc7c4f9 |
| `github.com/redis/go-redis/v9` | v9.21.0 | v9.23.0 |
| `github.com/shirou/gopsutil/v4` | v4.26.6 | v4.26.10 |
| `golang.org/x/crypto` | v0.54.0 | v0.58.0 |
| `golang.org/x/sync` | v0.22.0 | v0.24.0 |
| `gorm.io/driver/postgres` | v1.6.0 | v1.6.3 |

间接依赖包括 SQLite、pgx、MySQL 驱动、QUIC、OpenAPI、validator、protobuf 和 `golang.org/x/*` 等，完整版本见 `go.mod`。

Go 版本从 **1.26.0 升级到 1.27.3**。升级时已实际遇到 `github.com/gin-contrib/sse@v1.2.0` 要求 Go >= 1.26.10，原本的 Go 1.26.4 无法继续；本次通过 `go get go@latest` 选用了最新稳定工具链。引用此库的业务项目和构建镜像也需要满足 Go >= 1.27.3。现有 CI 使用 `go-version-file: go.mod`，会跟随该要求。

版本策略采用 Go 的 `@latest` 解析规则：有稳定标签时采用最新稳定标签，无标签的模块使用伪版本。校验了 `go.mod` 中全部 **105 个依赖**，没有发现当前模块路径下仍可升级的声明。依赖图另外存在 22 个有新版本的模块，它们未作为本项目的 require 声明保留，属于上游图中的依赖；没有为单纯追平这些版本增加无用的直接约束。切换到不同模块路径的大版本属于单独的 API 迁移，不包含在本次版本更新中。

后续代码优化移除了未使用的 SQLite 驱动依赖树及 `pkg/errors`，最终 `go.mod` 保留 **95 个依赖声明**。已再次查询上游版本，95 个声明在各自模块路径下均无剩余更新。

## 验证与边界

| 检查 | 结果 |
| --- | --- |
| 升级前 `go test ./...` | 通过 |
| 升级后 `go test -race -cover ./...` | 通过 |
| `go vet ./...` | 通过 |
| `go mod verify` | 通过，all modules verified |
| `go mod tidy` | 完成 |
| `go list -m -u -json all` 与 require 列表比对 | 105 个声明依赖均无剩余更新 |
| `govulncheck -show verbose ./...` | 当前调用及导入的包没有命中已知漏洞 |

漏洞工具另外报告了模块级公告 [GO-2026-5932](https://pkg.go.dev/vuln/GO-2026-5932)：`golang.org/x/crypto/openpgp` 已停止维护。项目实际使用的是 `bcrypt`，没有导入 `openpgp`。最新稳定版 `x/crypto v0.58.0` 仍包含该模块级公告；工具列出的修复版本为伪版本 `v0.58.1-0.20261009181718-2f1f834330b2`。本次保留最新稳定版。

验证环境为 macOS/arm64、Go 1.27.3，现有接口测试使用内存 SQLite。尚未连接真实 MySQL、PostgreSQL、Redis，未执行宿主业务应用的端到端测试。竞态检测只覆盖测试实际执行的路径，不能据此判断热更新等未覆盖的并发路径安全。

## 优先修复的问题

以下为优化前实现的问题记录。对应修复、测试与实现边界见实施说明。

### P1：分页查询的 OR 条件可以突破调用方的企业过滤（已复现）

位置：`services/index.go:162`，尤其是搜索条件的 `db.Or(...)`。

调用方使用 `WithDB(db.Where("company_guid = ?", "a"))` 注入企业约束，再调用 `List(..., "name,code")` 时，条件会形成类似 `(company_guid = 'a' AND name LIKE ?) OR code LIKE ?` 的逻辑。第二个搜索字段命中时，其他企业的数据也可能被返回。

临时内存数据库复现：企业 a 的角色 `name=needle`，企业 b 的角色 `code=needle`；在企业 a 的查询中搜索 `needle`，实际返回两家企业的角色，`total=2`。这是通用服务组合使用时的隔离缺陷，不代表所有现有 API 都调用了该方法。

建议把所有搜索 OR 条件构造成独立分组，再用 `db.Where(group)` 与企业条件相与，形成 `company_guid = ? AND (name LIKE ? OR code LIKE ?)`；增加“企业约束 + 第二搜索字段命中”的回归测试。

### P1：用户名唯一索引的迁移判断失效（已复现）

位置：`domains/user.go:11`、`services/identity_service.go:41`。

模型的 `gorm:"index"` 已经创建名为 `idx_sys_user_username` 的普通索引，迁移只通过 `HasIndex` 判断名称是否存在，因而跳过同名 UNIQUE 索引的创建。应用层的“先查重、再插入”不能阻止不同企业的并发创建。

临时内存数据库复现：执行 `AutoMigrate` 和 `InitializeIdentity` 后，连续插入两个 `Username="duplicate"`、GUID 不同的用户，两次均成功。

已在模型中直接声明唯一索引，已删除历史身份和旧索引兼容迁移逻辑。用户名统一小写、去除首尾空格，包含软删除账号在内保持全局唯一；数据库唯一冲突转换为业务错误。

### P1：AES 解密对非法输入会 panic，哈希函数丢弃错误（已复现部分）

位置：`utils/bcrypt_util.go:17`、`:59`、`:83`。

`AesDecrypt("")` 已复现 `index out of range [-1]`。非整块密文会在 `CryptBlocks` 中 panic；PKCS7 去填充没有校验长度和每个填充字节。HTTP 恢复中间件可能把它转换成 500，库的其他调用方也可能直接收到 panic。

建议在解密前检查空密文、分组长度和填充是否合法，统一返回 error；增加畸形输入和 fuzz 测试。`BcryptHash` 应返回 `(string, error)` 并传递 bcrypt 的密码长度等错误，防止静默保存空哈希。长期可通过带版本号的协议迁移到随机 nonce 的 AES-GCM；现有 CBC 使用固定密钥及固定 IV，不应作为可信的保密机制。

### P1：解压工具缺少完整的目录边界保护（代码检查）

位置：`utils/zip_util.go:13`、`:52`、`:103`。

`Unzip`、`UnzipWithCustomName` 直接拼接 ZIP 条目名；`../` 可以逃离目标目录。`UnzipFlatten` 使用字符串前缀判断边界，目标目录与同名前缀的相邻目录会混淆。宿主若将不可信 ZIP 交给这些函数，会存在覆盖目标目录外文件的风险；本次未确认在线接口直接调用这些工具。

建议统一使用 `os.OpenRoot` 限定文件系统访问范围，校验条目路径及符号链接，限制条目数和解压后的总字节数，并在每轮循环内关闭文件，避免循环末尾才统一执行 defer。

### P1：配置热更新的锁没有保护读路径（代码检查）

位置：`middlewares/viper.go:74`、`global/global.go`、`utils/jwt_util.go:33`、`middlewares/jwt.go`。

热更新持写锁替换整个 `global.NAV_CONFIG`，大量请求直接读取该结构却没有持读锁。写入方单独加锁不能避免数据竞争，同一请求也可能读到多个配置版本；Viper 本身的并发读写也需要统一约束。

建议通过 `atomic.Pointer[configs.Server]` 或 `atomic.Value` 发布不可变快照，每个请求读取一次；发布前执行与启动一致的配置校验。JWT 密钥、路由前缀、数据库等配置需要明确哪些可以热更新、哪些必须重启，并为并发请求加热更新补充竞态测试。

### P1：服务关闭顺序与资源生命周期不完整（代码检查）

位置：`inits/server.go:84`、`inits/system.go:88`、`utils/cache_util.go:17`。

当前在 `srv.Shutdown` 之前关闭 IP 查询资源并执行关闭钩子，此时仍有请求在执行，可能继续使用已释放资源。定时任务管理器仅保存在局部变量，缓存持久化 goroutine 没有取消入口；没有统一关闭数据库、Redis 和刷新 Zap，也没有 `signal.Stop` 或启动失败时的完整回滚。`OtherInit` 还忽略 IP 数据库初始化错误。

建议由应用实例持有资源和清理函数：先停止接收请求并等待执行中的请求完成，再停止后台任务、最终持久化缓存、释放数据库/Redis/IP 查询资源并刷新日志；为关闭钩子设置明确语义，为初始化失败提供反向清理。

### P1：SQLite 启动过程重复打开数据库并丢失连接配置（代码检查）

位置：`inits/system.go:39`、`services/db_service.go:57`、`dbs/gorm_sqlite.go:26`、`dbs/initdb_sqlite.go:54`。

启动先使用官方 SQLite 驱动设置 WAL、busy_timeout 和连接池，再通过初始化流程使用 `glebarez/sqlite` 重新打开并替换 `global.NAV_DB`，原连接池没有关闭，新连接池没有继承这些设置。busy_timeout 本身属于连接级配置，单次 Exec 也不能保证覆盖池中的全部连接。

建议初始化流程复用同一 `*gorm.DB`，统一选用一种 SQLite 驱动；使用驱动 DSN/连接初始化钩子设置每条连接的参数，检查 PRAGMA 和 `db.DB()` 错误，并根据写入模型设置连接池上限。`configs/sqlite.go:14` 总是附加 `.db`，README 的 `db-name: nav-app.db` 会生成 `nav-app.db.db`，配置约定也应一起统一。

## 性能与工程优化

### P2：Casbin 缓存时长单位写错

位置：`middlewares/casbin.go:88`、`:117`。

`SetExpireTime(60 * 60)` 接收 `time.Duration`，实际是 **3600 纳秒，即 3.6 微秒**，基本无法命中缓存。若原意为一小时，应改为 `time.Hour`。本项目同时使用实时 AccessContext 校验，修复缓存后仍应保持权限变更立即生效，并为多实例刷新设计 Redis 通知或策略版本机制。

### P2：日志写入热路径执行过多文件操作

位置：`middlewares/zap.go:127`、`:178`。

每条日志都执行目录检查、打开文件、写入和关闭；启用 retention 后还会对整个日志目录执行 Walk。这些操作在锁中串行完成，日志量增加时会放大磁盘和锁竞争。

建议复用文件句柄，在轮转时重新打开；把过期清理移到每日任务，可使用成熟的轮转组件或标准输出交由部署平台收集。用并发写入基准测试比较吞吐和分配，避免只凭实现推断具体收益。

### P2：审计日志应限制请求体、脱敏并正确收集错误

位置：`middlewares/logger.go:28`、`:71`、`:80`、`:96`。

ApiLogger 无上限读取请求体并把原始参数写入数据库；用户管理等接口的密码密文也会被保存，而其加密密钥是固定值。错误在 `c.Next()` 之前收集，会漏掉处理器产生的错误；IP 查询与日志数据库写入同步增加请求耗时。上传实现也缺少统一请求大小上限。

建议按接口和内容类型设置 `http.MaxBytesReader`，日志只截取有限长度，对密码、令牌等字段脱敏，在 `c.Next()` 后提取错误和响应状态。对允许异步的日志使用有容量上限的队列，明确队列满时的处理和停机 drain；必须可靠保存的审计事件可使用事务 outbox。

### P2：CORS 配置和 Cookie 属性应实际生效

位置：`inits/server.go:119`、`middlewares/cors.go`、`utils/jwt_util.go:165`。

默认路由始终使用 `Cors()`，直接反射任意 Origin 并允许凭据，已有 `CorsByRules()` 配置没有接入。JWT Cookie 的 Secure、HttpOnly 均为 false，也没有显式 SameSite 策略。

建议路由接入配置化的 CORS 白名单、补充 `Vary: Origin`，按部署方式设置 Cookie 的 Secure/HttpOnly/SameSite。若前端读取现有 Cookie，需要先约定认证方式再迁移。初始化 Gin 时也应配置可信代理范围，使 IP 审计使用的 `ClientIP()` 符合实际反向代理拓扑。

### P2：认证查询可减少数据库负担

位置：`services/access_service.go:55`、`services/identity_service.go:18`。

普通用户每次认证通常查询用户、企业、角色及权限，需保留实时身份与授权校验。优化前的身份初始化还会在启动时全量读取、更新历史用户；现已删除这一历史处理。

当前已添加相关索引、规范化账号后使用索引等值查询，并把用户列表的角色查询合并为一次批量查询。认证仍保留实时停用、企业状态与会话失效校验。若后续引入权限数据缓存，缓存键应覆盖企业、用户及授权版本，不能仅依赖 JWT 内的旧角色信息。

### P3：减少全局状态和基础库的默认装配

位置：`global/global.go`、`inits/server.go:17`、`inits/inits.go`。

逐步引入持有 DB、Redis、Logger、配置快照和后台任务的 App 实例，通过构造函数注入服务，兼容保留现有入口，可降低测试间相互影响并允许同进程多个实例。Swagger 当前绑定上游 `example/basic/docs`，应改为宿主注册文档并可关闭入口。`OnCacheInit` 目前仅有字段和类型，缺少对外注册方法和实际执行，应补全或修正文档。仅使用 `errors.New` 的代码可迁回标准库，逐步减少 `pkg/errors` 等不必要依赖。

## 建议推进顺序

1. 修复已经复现的企业搜索隔离、用户名唯一索引和 AES 非法输入问题，同时加入对应回归测试。
2. 处理解压目录边界、配置快照、SQLite 单连接入口及可靠关闭流程。
3. 修正 Casbin 时长，优化日志文件轮转、请求体上限、脱敏和 CORS/Cookie 配置。
4. 在真实 MySQL、PostgreSQL、Redis 环境增加集成测试；CI 纳入竞态检测、依赖漏洞扫描和依赖更新检查，再开展认证查询基准测试与 App 实例重构。

优化前测试集中在 `middlewares/access_test.go` 的接口与身份场景。本次已增加分页企业隔离、唯一约束、热更新并发、完整 SQLite 初始化关闭、启动回滚、上传限制、解压边界等行为测试，具体见实施说明。
