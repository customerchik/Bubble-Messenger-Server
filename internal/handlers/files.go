package handlers

import (
	"bubble/internal/crypto"
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

type SendFileRequest struct {
	Receiver string
	FileName string
	File     []byte
}

type DelFileRequest struct {
	Filename string
}

type SetAvatarRequest struct {
	Avatar []byte `json:"avatar"`
}

type GetAvatarRequest struct {
	LoginForSearch string
}

type GetFileRequest struct {
	StorageFilepath string
}

func (h *Handler) SendFile(c echo.Context, requestBody []byte, key string, userLogin string) error {
	var req SendFileRequest
	if err := json.Unmarshal(requestBody, &req); err != nil {
		h.Blocker.RegisterStrike(c.RealIP())
		return c.JSON(http.StatusBadRequest, "Invalid JSON")
	}

	if req.Receiver == "" || req.FileName == "" || req.File == nil {
		h.Blocker.RegisterStrike(c.RealIP())
		return c.String(http.StatusBadRequest, "Did not receive all server data")
	}

	statusCh := make(chan string)
	go h.FilesService.SaveFileTempPath(c.Request().Context(), statusCh)

	tempPath := <-statusCh
	if strings.Contains(tempPath, "Error") {
		return c.NoContent(http.StatusInternalServerError)
	} else {
		if _, err := c.Response().Write([]byte(tempPath + "\n")); err != nil {
			return err
		}

		c.Response().Flush()
	}

	fileStoragePath := <-statusCh

	if err := h.MessengerService.SendFile(c.Request().Context(), h.UsersService, userLogin, req.Receiver, req.FileName, fileStoragePath); err != nil {
		go h.FilesService.DelFile(context.Background(), fileStoragePath)
		c.Logger().Errorf("Send file ERR: %s", err)
		resp := err.Error()
		if key != "" {
			resp, _ = crypto.StringEncrypt([]byte(resp), key)
		}
		return c.String(http.StatusInternalServerError, resp)
	}

	return c.NoContent(http.StatusOK)
}

func (h *Handler) DelFile(c echo.Context, requestBody []byte, key string) error {
	var req DelFileRequest
	if err := json.Unmarshal(requestBody, &req); err != nil {
		h.Blocker.RegisterStrike(c.RealIP())
		return c.JSON(http.StatusBadRequest, "Invalid JSON")
	}

	if req.Filename == "" {
		h.Blocker.RegisterStrike(c.RealIP())
		return c.String(http.StatusBadRequest, "Did not receive all server data")
	}

	if err := h.FilesService.DelFile(c.Request().Context(), req.Filename); err != nil {
		c.Logger().Error("Del file ERR: %s", err)
		resp := "Cant del file"
		if key != "" {
			resp, _ = crypto.StringEncrypt([]byte(resp), key)
		}
		return c.String(http.StatusInternalServerError, resp)
	}

	return c.NoContent(http.StatusOK)
}

func (h *Handler) SetAvatar(c echo.Context, requestBody []byte, key string, userLogin string) error {
	var req SetAvatarRequest
	if err := json.Unmarshal(requestBody, &req); err != nil {
		h.Blocker.RegisterStrike(c.RealIP())
		return c.JSON(http.StatusBadRequest, "Invalid JSON")
	}

	statusCh := make(chan string)
	go h.FilesService.SaveAvatarTempPath(c.Request().Context(), statusCh)

	tempPath := <-statusCh
	if strings.Contains(tempPath, "Error") {
		return c.NoContent(http.StatusInternalServerError)
	} else {
		if _, err := c.Response().Write([]byte(tempPath + "\n")); err != nil {
			return err
		}

		c.Response().Flush()
	}

	storageAvatarPath := <-statusCh

	if err := h.UsersService.SetAvatar(c.Request().Context(), userLogin, storageAvatarPath); err != nil {
		c.Logger().Error("Set avatar ERR %s", err)
		resp := err.Error()
		if key != "" {
			resp, _ = crypto.StringEncrypt([]byte(resp), key)
		}
		return c.String(http.StatusBadRequest, resp)
	}

	c.Logger().Infof("Success save avatar by %s", userLogin)

	return c.NoContent(http.StatusOK)
}

func (h *Handler) GetAvatar(c echo.Context, requestBody []byte, key string) error {
	var req GetAvatarRequest
	if err := json.Unmarshal(requestBody, &req); err != nil {
		h.Blocker.RegisterStrike(c.RealIP())
		return c.JSON(http.StatusBadRequest, "Invalid JSON")
	}

	if req.LoginForSearch == "" {
		h.Blocker.RegisterStrike(c.RealIP())
		return c.String(http.StatusBadRequest, "Did not receive all server data")
	}

	storagePath, err := h.UsersService.GetAvatar(c.Request().Context(), req.LoginForSearch)
	if err != nil {
		c.Logger().Error("Get Avatar ERR: %s", err)
		resp := err.Error()
		if key != "" {
			resp, _ = crypto.StringEncrypt([]byte(resp), key)
		}
		return c.String(http.StatusInternalServerError, resp)
	}

	tempPath, err := h.FilesService.DownloadFileTempPath(c.Request().Context(), storagePath)
	if err != nil {
		resp := "Cant generate temp path"
		if key != "" {
			resp, _ = crypto.StringEncrypt([]byte(resp), key)
		}
		return c.String(http.StatusInternalServerError, resp)
	}

	if key != "" {
		storagePath, err = crypto.StringEncrypt([]byte(tempPath), key)
		if err != nil {
			resp := "Decrypted error"
			if key != "" {
				resp, _ = crypto.StringEncrypt([]byte(resp), key)
			}
			return c.String(http.StatusInternalServerError, resp)
		}
	}

	return c.String(http.StatusOK, tempPath)
}

func (h *Handler) GetFile(c echo.Context, requestBody []byte, key string, userLogin string) error {
	var req GetFileRequest
	if err := json.Unmarshal(requestBody, &req); err != nil {
		h.Blocker.RegisterStrike(c.RealIP())
		return c.JSON(http.StatusBadRequest, "Invalid JSON")
	}

	if req.StorageFilepath == "" {
		h.Blocker.RegisterStrike(c.RealIP())
		return c.String(http.StatusBadRequest, "Did not receive all server data")
	}

	if strings.Split(req.StorageFilepath, ".")[0] != userLogin {
		resp := "Not access"
		if key != "" {
			resp, _ = crypto.StringEncrypt([]byte(resp), key)
		}
		return c.String(http.StatusInternalServerError, resp)
	}

	tempPath, err := h.FilesService.DownloadFileTempPath(c.Request().Context(), req.StorageFilepath)
	if err != nil {
		resp := "Cant generate temp path"
		if key != "" {
			resp, _ = crypto.StringEncrypt([]byte(resp), key)
		}
		return c.String(http.StatusInternalServerError, resp)
	}

	if key != "" {
		tempPath, _ = crypto.StringEncrypt([]byte(tempPath), key)
	}
	return c.String(http.StatusOK, tempPath)
}
