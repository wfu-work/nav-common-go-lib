package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Code int         `json:"code"`
	Data interface{} `json:"data"`
	Msg  string      `json:"msg"`
}

func Result(code int, data interface{}, msg string, c *gin.Context) {
	c.JSON(http.StatusOK, Response{
		code,
		data,
		msg,
	})
}

func Ok(data interface{}, c *gin.Context) {
	Result(http.StatusOK, data, "请求成功", c)
}

func OkWithMessage(message string, c *gin.Context) {
	Result(http.StatusOK, map[string]interface{}{}, message, c)
}

func OkWithData(data interface{}, c *gin.Context) {
	Result(http.StatusOK, data, "请求成功", c)
}

func OkWithDetailed(data interface{}, message string, c *gin.Context) {
	Result(http.StatusOK, data, message, c)
}

func Fail(data interface{}, c *gin.Context) {
	Result(http.StatusBadRequest, data, "请求失败", c)
}

func FailWithMessage(message string, c *gin.Context) {
	Result(http.StatusBadRequest, map[string]interface{}{}, message, c)
}

func FailWithDetailed(data interface{}, message string, c *gin.Context) {
	Result(http.StatusBadRequest, data, message, c)
}

func NoAuth(message string, c *gin.Context) {
	c.JSON(http.StatusUnauthorized, Response{
		http.StatusUnauthorized,
		nil,
		message,
	})
}

func NoPermission(message string, c *gin.Context) {
	c.JSON(http.StatusForbidden, Response{
		http.StatusForbidden,
		nil,
		message,
	})
}
