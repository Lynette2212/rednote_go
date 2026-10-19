package common

import (
	"rednote/internal/models"

	"github.com/gin-gonic/gin"
)

const ctxUserKey = "currentUser"

func SetCurrentUser(ctx *gin.Context, user *models.User) {
	ctx.Set(ctxUserKey, user)
}

func GetCurrentUser(ctx *gin.Context) *models.User {
	if v, ok := ctx.Get(ctxUserKey); ok {
		if user, ok := v.(*models.User); ok {
			return user
		}
	}
	return nil
}
