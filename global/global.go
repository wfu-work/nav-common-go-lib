package global

import (
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/patrickmn/go-cache"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"
	"gorm.io/gorm"
	"navfirst.com/nav-common-go-lib/configs"

	_ "github.com/gin-gonic/gin"

	"github.com/spf13/viper"
)

var (
	NAV_DB                  *gorm.DB
	NAV_REDIS               redis.UniversalClient
	NAV_CONFIG              configs.Server
	NAV_VIPER               *viper.Viper
	NAV_LOG                 *zap.Logger
	NAV_Concurrency_Control = &singleflight.Group{}
	NAV_ROUTERS             gin.RoutesInfo
	NAV_ACTIVE_DBNAME       *string
	LOCAL_CACHE             *cache.Cache
	Lock                    sync.RWMutex
)
