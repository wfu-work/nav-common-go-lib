package inits

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/wfu-work/nav-common-go-lib/global"
	"github.com/wfu-work/nav-common-go-lib/middlewares"
	"github.com/wfu-work/nav-common-go-lib/routers"
)

type Server interface {
	ListenAndServe() error
	Shutdown(context.Context) error
}

func (s *SysInit) RunServer() (err error) {
	defer func() { err = errors.Join(err, s.Close()) }()
	Router := s.Routers()

	address := fmt.Sprintf(":%d", global.Config().System.Addr)
	routerPrefix := global.Config().System.RouterPrefix
	appName := global.Config().System.AppName

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
		Addr:              address,
		Handler:           router,
		ReadTimeout:       readTimeout,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       60 * time.Second,
		WriteTimeout:      writeTimeout,
		MaxHeaderBytes:    1 << 20,
	}
	parent := s.background
	if parent == nil {
		parent = context.Background()
	}
	ctx, stop := signal.NotifyContext(parent, syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	return runServer(ctx, srv)
}

func runServer(ctx context.Context, srv Server) error {
	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()
	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
	}
	drain, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(drain); err != nil {
		if closer, ok := srv.(interface{ Close() error }); ok {
			_ = closer.Close()
		}
		return err
	}
	err := <-errCh
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

// Routers 初始化总路由
func (s *SysInit) Routers() *gin.Engine {
	Router := gin.New()
	Router.Use(gin.Recovery(), middlewares.RequestLimits())
	if err := Router.SetTrustedProxies(global.Config().System.TrustedProxies); err != nil {
		panic(err)
	}
	if gin.Mode() == gin.DebugMode {
		Router.Use(gin.Logger())
	}

	systemRouter := routers.SysRouterGroupApp

	if s.onWebInit != nil {
		s.onWebInit(Router)
	}

	Router.Use(middlewares.CorsByRules())
	log.Println("use middleware cors")

	if global.Config().System.EnableSwagger {
		// Hosts register their own generated docs, usually with a blank import.
		Router.GET(global.Config().System.RouterPrefix+"/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	PublicGroup := Router.Group(global.Config().System.RouterPrefix)
	{
		PublicGroup.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, "ok")
		})
		PublicGroup.GET(global.Config().Local.ProxyOssPath+"/*filepath", middlewares.JWTAuth(), middlewares.CasbinHandler(), gin.WrapH(
			http.StripPrefix(global.Config().System.RouterPrefix+global.Config().Local.ProxyOssPath+"/", http.FileServer(http.Dir(global.Config().Local.OssPath))),
		))
	}
	PrivateGroup := Router.Group(global.Config().System.RouterPrefix)
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
