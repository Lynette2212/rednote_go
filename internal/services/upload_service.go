package services

import (
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"rednote/internal/pkg/config"
	"rednote/internal/pkg/errs"

	"github.com/google/uuid"
)

var UploadService = newUploadService()

func newUploadService() *uploadService {
	return &uploadService{}
}

type uploadService struct {
}

var allowedImageExts = map[string]string{
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".gif":  "image/gif",
	".webp": "image/webp",
}

// 由文件内容嗅探出的 MIME → 落盘扩展名（扩展名以内容为准，不信用户文件名）
var allowedImageMIMEs = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/gif":  ".gif",
	"image/webp": ".webp",
}

const MaxImageUploadSize = 20 << 20 // 20MB

// SaveImage 保存上传的图片，返回可访问的 URL 路径
func (s *uploadService) SaveImage(file multipart.File, header *multipart.FileHeader) (string, error) {
	if header.Size > MaxImageUploadSize {
		return "", errs.ParamsError("图片不能超过 20MB")
	}
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if _, ok := allowedImageExts[ext]; !ok {
		return "", errs.ParamsError("仅支持 jpg/png/gif/webp 图片")
	}

	// 只看扩展名不够（把 .txt 改名成 .jpg 就能骗过），读前 512 字节嗅探真实类型
	head := make([]byte, 512)
	n, err := io.ReadFull(file, head)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		return "", err
	}
	if n == 0 {
		return "", errs.ParamsError("文件内容为空")
	}
	realExt, ok := allowedImageMIMEs[http.DetectContentType(head[:n])]
	if !ok {
		return "", errs.ParamsError("文件内容不是有效图片")
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", err
	}

	now := time.Now()
	relDir := filepath.Join("images", now.Format("2006"), now.Format("01"), now.Format("02"))
	absDir := filepath.Join(config.Instance.UploadDir, relDir)
	if err := os.MkdirAll(absDir, 0o755); err != nil {
		return "", err
	}
	filename := uuid.NewString() + realExt
	absPath := filepath.Join(absDir, filename)

	dst, err := os.Create(absPath)
	if err != nil {
		return "", err
	}
	// header.Size 由客户端提供，不可信；用 LimitReader 强制上限，避免超大文件占满磁盘
	written, err := io.Copy(dst, io.LimitReader(file, MaxImageUploadSize+1))
	if err != nil {
		dst.Close()
		os.Remove(absPath)
		return "", err
	}
	if written > MaxImageUploadSize {
		dst.Close()
		os.Remove(absPath)
		return "", errs.ParamsError("图片不能超过 20MB")
	}
	if err := dst.Close(); err != nil {
		os.Remove(absPath)
		return "", err
	}
	return "/uploads/" + filepath.ToSlash(relDir) + "/" + filename, nil
}
