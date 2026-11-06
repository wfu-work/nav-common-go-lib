// Package apis 提供基础库的 Gin HTTP 处理器。
//
// 路径与文档生成
//
// Swagger 的 @Router 路径相对于 system.router-prefix，默认前缀为 /api。
// 宿主负责注册路由、导入生成的文档，并在 Swagger 主入口配置 BasePath 和
// ApiKeyAuth（header: Authorization，值为 Bearer <token>）。导航及菜单接口
// 由 routers.InitNavigationRouter 单独挂载，宿主还须注册导航目录。
//
// 认证与企业范围
//
// 默认公开入口为登录、验证码和加密；其他入口按默认路由链验证 JWT 和权限，
// 当前用户、当前企业摘要、修改自身密码及退出登录只要求有效登录身份。
// 平台管理员指固定内置 admin 身份；普通企业用户只能在接口明确支持的企业
// 范围内操作。具体管理员限制、companyGuid 规则见各处理器说明。
//
// 响应约定
//
// 除成功下载文件直接返回二进制流外，接口使用 response.Response：code、data、msg。
// 成功通常为 HTTP 200、code=200；参数或业务失败通常仍为 HTTP 200、code=400，
// 客户端须检查 code 和 msg。认证失败使用 HTTP 401，权限失败使用 HTTP 403。
// 请求体超限可由全局中间件返回 HTTP 413，无 JSON 响应体；普通请求默认上限
// 16 MiB，上传请求默认上限 32 MiB，包含编码开销。
//
// 分页结果位于 Response.data，其中包含 data 列表、total、page、size。
// 时间戳通常为 Unix 毫秒；分页参数、搜索字段、排序及敏感字段裁剪以各接口
// 注释为准。停用的注册、企业删除入口会明确返回业务失败；未注册的处理器
// 不声明 Swagger 路由。
package apis
