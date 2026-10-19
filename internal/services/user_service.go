package services

import (
	"strings"
	"time"

	"rednote/internal/models"
	"rednote/internal/pkg/errs"
	"rednote/internal/pkg/sqls"
	"rednote/internal/repositories"

	"golang.org/x/crypto/bcrypt"
)

var UserService = newUserService()

func newUserService() *userService {
	return &userService{}
}

type userService struct {
}

func (s *userService) Get(id int64) *models.User {
	return repositories.UserRepository.Get(sqls.DB(), id)
}

func (s *userService) GetByUsername(username string) *models.User {
	return repositories.UserRepository.FindOne(sqls.DB(), sqls.NewCnd().Eq("username", username))
}

func (s *userService) SignUp(username, nickname, password string) (*models.User, error) {
	username = strings.TrimSpace(username)
	nickname = strings.TrimSpace(nickname)
	if len(username) < 3 || len(username) > 32 {
		return nil, errs.ParamsError("用户名长度需在 3-32 之间")
	}
	if len(password) < 6 {
		return nil, errs.ParamsError("密码长度至少 6 位")
	}
	if nickname == "" {
		nickname = username
	}
	if s.GetByUsername(username) != nil {
		return nil, errs.ParamsError("用户名已存在")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	now := time.Now().UnixMilli()
	user := &models.User{
		Username:   username,
		Nickname:   nickname,
		Password:   string(hash),
		Avatar:     "",
		CreateTime: now,
		UpdateTime: now,
	}
	if err := repositories.UserRepository.Create(sqls.DB(), user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *userService) SignIn(username, password string) (*models.User, error) {
	user := s.GetByUsername(strings.TrimSpace(username))
	if user == nil {
		return nil, errs.ParamsError("用户名或密码错误")
	}
	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)) != nil {
		return nil, errs.ParamsError("用户名或密码错误")
	}
	return user, nil
}

func (s *userService) UpdateProfile(userId int64, nickname, avatar, bio string, gender int) error {
	columns := map[string]interface{}{
		"nickname":   strings.TrimSpace(nickname),
		"avatar":     avatar,
		"bio":        strings.TrimSpace(bio),
		"gender":     gender,
		"update_time": time.Now().UnixMilli(),
	}
	return repositories.UserRepository.Updates(sqls.DB(), userId, columns)
}

func (s *userService) GetUsersByIds(ids []int64) map[int64]*models.User {
	ret := make(map[int64]*models.User)
	if len(ids) == 0 {
		return ret
	}
	list := repositories.UserRepository.Find(sqls.DB(), sqls.NewCnd().In("id", ids))
	for i := range list {
		ret[list[i].Id] = &list[i]
	}
	return ret
}
