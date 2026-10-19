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
	payload := buildJSONResult(result)
	ctx.JSON(httpStatusFromCode(payload.Code), payload)
}

// httpStatusFromCode 把业务 code 映射为 HTTP 状态，便于代理/监控/客户端正确识别
func httpStatusFromCode(code int) int {
	switch code {
	case errs.CodeSuccess:
		return http.StatusOK
	case errs.CodeParamsError, errs.CodeNotLogin, errs.CodeNoPermission,
		errs.CodeNotFound, errs.CodeTooManyRequests, errs.CodeInternalError:
		return code
	default:
		if code >= 400 && code < 600 {
			return code
		}
		return http.StatusInternalServerError
	}
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
