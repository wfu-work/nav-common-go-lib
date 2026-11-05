package routers

import (
	"github.com/gin-gonic/gin"
	"github.com/wfu-work/nav-common-go-lib/middlewares"
)

type CompanyRouter struct{}

func (s *CompanyRouter) InitCompanyRouter(Router *gin.RouterGroup) {
	routerLogger := Router.Group("company").Use(middlewares.ApiLogger())
	router := Router.Group("company")
	{
		routerLogger.POST("", companyApi.Save)                // 新增企业
		routerLogger.DELETE(":guid", companyApi.DeleteByGuid) // 删除企业
		routerLogger.PUT("pass/:guid", companyApi.Pass)       // 审核通过
		routerLogger.PUT("refuse/:guid", companyApi.Refuse)   // 审核拒绝
	}
	{
		router.GET("current", companyApi.Current)
		routerLogger.PUT("current", companyApi.UpdateCurrent)
		router.GET(":guid", companyApi.GetByGuid)  // 查询企业
		router.GET("list", companyApi.List)        // 分页获取企业列表
		router.GET("list/all", companyApi.ListAll) // 获取所有企业列表
	}
}
