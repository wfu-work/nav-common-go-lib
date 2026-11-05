package inits

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/swaggo/swag/example/basic/docs"
	"github.com/wfu-work/nav-common-go-lib/global"
	"github.com/wfu-work/nav-common-go-lib/middlewares"
	"github.com/wfu-work/nav-common-go-lib/routers"
	"github.com/wfu-work/nav-common-go-lib/utils"
	"github.com/wfu-work/nav-common-go-lib/utils/ip2geo"
	"go.uber.org/zap"
)

type Server interface {
	ListenAndServe() error
	Shutdown(context.Context) error
}

func (s *SysInit) RunServer() error {
	Router := s.Routers()

	address := fmt.Sprintf(":%d", global.NAV_CONFIG.System.Addr)
	routerPrefix := global.NAV_CONFIG.System.RouterPrefix
	appName := global.NAV_CONFIG.System.AppName

	fmt.Printf(`
	欢迎使用 %s
	当前版本: v1.0.0
	加群方式: 微信号：navfirst
	官方网址: https://www.navfirst.com
	默认后端运行地址: http://127.0.0.1%s%s
	默认自动化文档地址: http://127.0.0.1%s%s/swagger/index.html
	默认前端文件运行地址: http://127.0.0.1%s
	--------------------------------------版权声明--------------------------------------
	** 版权所有方：守明团队 **
	** 版权持有公司：武汉守明科技有限公司 **
`, appName, address, routerPrefix, address, routerPrefix, address)

	return s.initServer(address, Router, 10*time.Minute, 10*time.Minute)
}

// initServer 启动服务并实现优雅关闭
func (s *SysInit) initServer(address string, router *gin.Engine, readTimeout, writeTimeout time.Duration) error {
	srv := &http.Server{
		Addr:           address,
		Handler:        router,
		ReadTimeout:    readTimeout,
		WriteTimeout:   writeTimeout,
		MaxHeaderBytes: 1 << 20,
	}
	errCh := make(chan error, 1)

	// 在goroutine中启动服务
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	// 等待中断信号以优雅地关闭服务器
	quit := make(chan os.Signal, 1)
	// kill (无参数) 默认发送 syscall.SIGTERM
	// kill -2 发送 syscall.SIGINT
	// kill -9 发送 syscall.SIGKILL，但是无法被捕获，所以不需要添加
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	select {
	case err := <-errCh:
		zap.L().Error("server启动失败", zap.Error(err))
		return err
	case <-quit:
	}
	log.Println("关闭WEB服务...")

	utils.PersistCache()
	ip2geo.CloseIpGeoLocation()

	if s.onShutInit != nil {
		s.onShutInit()
	}

	// 设置5秒的超时时间
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		return err
	}
	log.Println("WEB服务已关闭")
	return nil
}

// Routers 初始化总路由
func (s *SysInit) Routers() *gin.Engine {
	Router := gin.New()
	Router.Use(gin.Recovery())
	if gin.Mode() == gin.DebugMode {
		Router.Use(gin.Logger())
	}

	systemRouter := routers.SysRouterGroupApp

	//Router.StaticFS(global.NAV_CONFIG.Local.StorePath, justFilesFilesystem{http.Dir(global.NAV_CONFIG.Local.StorePath)}) // Router.Use(middleware.LoadTls())  // 如果需要使用https 请打开此中间件 然后前往 core/server.go 将启动模式 更变为 Router.RunTLS("端口","你的cre/pem文件","你的key文件")

	if s.onWebInit != nil {
		s.onWebInit(Router)
	}

	Router.Use(middlewares.Cors())
	log.Println("use middleware cors")

	docs.SwaggerInfo.BasePath = global.NAV_CONFIG.System.RouterPrefix
	Router.GET(global.NAV_CONFIG.System.RouterPrefix+"/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	log.Println("register swagger handler")

	PublicGroup := Router.Group(global.NAV_CONFIG.System.RouterPrefix)
	{
		PublicGroup.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, "ok")
		})
		PublicGroup.GET(global.NAV_CONFIG.Local.ProxyOssPath+"/*filepath", middlewares.JWTAuth(), middlewares.CasbinHandler(), gin.WrapH(
			http.StripPrefix(global.NAV_CONFIG.System.RouterPrefix+global.NAV_CONFIG.Local.ProxyOssPath+"/", http.FileServer(http.Dir(global.NAV_CONFIG.Local.OssPath))),
		))
	}
	PrivateGroup := Router.Group(global.NAV_CONFIG.System.RouterPrefix)
	PrivateGroup.Use(middlewares.JWTAuth()).Use(middlewares.CasbinHandler())
	{
		systemRouter.InitCompanyRouter(PrivateGroup)            // 注册企业相关路由
		systemRouter.InitUserRouter(PrivateGroup)               // 注册用户相关路由
		systemRouter.InitRoleRouter(PrivateGroup)               // 注册系统角色路由
		systemRouter.InitPermissionRouter(PrivateGroup)         // 注册系统权限路由
		systemRouter.InitLogRouter(PrivateGroup)                // 注册日志相关路由
		systemRouter.InitOsRouter(PrivateGroup)                 // 注册服务器相关路由
		systemRouter.InitConfigRouter(PrivateGroup)             // 注册配置路由
		systemRouter.InitLoginRouter(PrivateGroup, PublicGroup) // 注册登录路由
		systemRouter.InitLoginLogRouter(PrivateGroup)           // 注册登录日志路由
		systemRouter.InitFileRouter(PrivateGroup)               // 注册文件路由
		systemRouter.InitSecretRouter(PrivateGroup)             // 注册加密路由
		systemRouter.InitRegisterRouter(PrivateGroup)           // 注册用户注册路由
	}

	{
		if s.onRouterInit != nil {
			s.onRouterInit(PublicGroup, PrivateGroup)
		}
	}

	// 注册业务路由
	initBizRouter(PrivateGroup, PublicGroup)

	global.NAV_ROUTERS = Router.Routes()

	log.Println("router register success")
	return Router
}

// 占位方法，保证文件可以正确加载，避免go空变量检测报错，请勿删除。
func holder(routers ...*gin.RouterGroup) {
	_ = routers
}

func initBizRouter(routers ...*gin.RouterGroup) {
	privateGroup := routers[0]
	publicGroup := routers[1]

	holder(publicGroup, privateGroup)
}
