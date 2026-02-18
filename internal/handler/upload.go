package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// UploadHandler 文件上传处理器
type UploadHandler struct {
	uploadDir string // 图片上传目录
}

// NewUploadHandler 创建上传处理器
func NewUploadHandler(uploadDir string) *UploadHandler {
	// 确保上传目录存在
	os.MkdirAll(uploadDir, 0755)
	return &UploadHandler{uploadDir: uploadDir}
}

// UploadImage 处理图片上传
func (h *UploadHandler) UploadImage(w http.ResponseWriter, r *http.Request) {
	// 限制上传大小为 10MB
	r.ParseMultipartForm(10 << 20)

	file, header, err := r.FormFile("image")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error": "请选择要上传的图片",
		})
		return
	}
	defer file.Close()

	// 验证文件类型
	contentType := header.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image/") {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error": "仅支持图片文件",
		})
		return
	}

	// 生成唯一文件名
	ext := filepath.Ext(header.Filename)
	if ext == "" {
		ext = ".png"
	}
	filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	filePath := filepath.Join(h.uploadDir, filename)

	// 保存文件
	dst, err := os.Create(filePath)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"error": "保存图片失败",
		})
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"error": "保存图片失败",
		})
		return
	}

	// 返回图片 URL
	url := "/Pic/" + filename

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"url": url,
	})
}
