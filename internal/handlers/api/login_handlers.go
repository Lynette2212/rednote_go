package api

import (
	"rednote/internal/models/req"
	"rednote/internal/pkg/common"
	"rednote/internal/pkg/errs"
	"rednote/internal/pkg/ginx"
	"rednote/internal/pkg/token"
	"rednote/internal/services"

	"github.com/gin-gonic/gin"
)

// Register 注册
func Register(ctx *gin.Context) {
	var r req.RegisterReq
	if err := ginx.Bind(ctx, &r); err != nil {
		ginx.WriteJSON(ctx, err)
		return
	}
	user, err := services.UserService.SignUp(r.Username, r.Nickname, r.Password)
	if err != nil {
		ginx.WriteJSON(ctx, err)
		return
	}
	tokenStr, err := token.Generate(user.Id)
	if err != nil {
		ginx.WriteJSON(ctx, err)
		return
	}
	ginx.WriteJSON(ctx, gin.H{"token": tokenStr, "user": user})
}

// Login 登录
func Login(ctx *gin.Context) {
	var r req.LoginReq
	if err := ginx.Bind(ctx, &r); err != nil {
		ginx.WriteJSON(ctx, err)
		return
	}
	user, err := services.UserService.SignIn(r.Username, r.Password)
	if err != nil {
		ginx.WriteJSON(ctx, err)
		return
	}
	tokenStr, err := token.Generate(user.Id)
	if err != nil {
		ginx.WriteJSON(ctx, err)
		return
	}
	ginx.WriteJSON(ctx, gin.H{"token": tokenStr, "user": user})
}

// UserInfo 当前登录用户信息
func UserInfo(ctx *gin.Context) {
	user := common.GetCurrentUser(ctx)
	if user == nil {
		ginx.WriteJSON(ctx, errs.NotLogin())
		return
	}
	ginx.WriteJSON(ctx, user)
}
