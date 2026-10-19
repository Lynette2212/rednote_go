package ginx

import (
	"net/http"

	"rednote/internal/pkg/errs"
	"rednote/internal/pkg/sqls"

	"github.com/gin-gonic/gin"
)

type JsonResult struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
	Success bool        `json:"success"`
}

type pageData struct {
	results interface{}
	paging  *sqls.Paging
}

func PageData(results interface{}, paging *sqls.Paging) interface{} {
	return pageData{results: results, paging: paging}
}

func Bind(ctx *gin.Context, obj interface{}) error {
	if err := ctx.ShouldBind(obj); err != nil {
		return errs.ParamsError(err.Error())
	}
	return nil
}

func WriteJSON(ctx *gin.Context, result interface{}) {
	ctx.JSON(http.StatusOK, buildJSONResult(result))
}

func buildJSONResult(result interface{}) *JsonResult {
	switch value := result.(type) {
	case nil:
		return &JsonResult{Code: errs.CodeSuccess, Success: true}
	case *JsonResult:
		return value
	case *errs.CodeError:
		return &JsonResult{Code: value.Code, Message: value.Message, Success: false}
	case error:
		ce := errs.FromError(value)
		return &JsonResult{Code: ce.Code, Message: ce.Message, Success: false}
	case pageData:
		return &JsonResult{
			Code:    errs.CodeSuccess,
			Success: true,
			Data:    gin.H{"results": value.results, "paging": value.paging},
		}
	default:
		return &JsonResult{Code: errs.CodeSuccess, Data: value, Success: true}
	}
}
