package api

import (
	"rednote/internal/pkg/common"
	"rednote/internal/pkg/errs"
	"rednote/internal/pkg/ginx"
	"rednote/internal/services"

	"github.com/gin-gonic/gin"
)

// Upload 上传图片，返回 {url}
func Upload(ctx *gin.Context) {
	user := common.GetCurrentUser(ctx)
	if user == nil {
		ginx.WriteJSON(ctx, errs.NotLogin())
		return
	}
	file, header, err := ctx.Request.FormFile("file")
	if err != nil {
		ginx.WriteJSON(ctx, errs.ParamsError("请选择文件"))
		return
	}
	defer file.Close()

	url, err := services.UploadService.SaveImage(file, header)
	if err != nil {
		ginx.WriteJSON(ctx, err)
		return
	}
	ginx.WriteJSON(ctx, gin.H{"url": url})
}
