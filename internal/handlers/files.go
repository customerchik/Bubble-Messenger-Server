package handlers

import (
	"bubble/internal/crypto"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

type SendFileRequest struct {
	Receiver string
	FileName string
}

type DelFileRequest struct {
	Filename string
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

	if req.Receiver == "" || req.FileName == "" {
		h.Blocker.RegisterStrike(c.RealIP())
		return c.String(http.StatusBadRequest, "Did not receive all server data")
	}

	statusCh := make(chan string)
	go h.FilesService.SaveFileTempPath(c.Request().Context(), key, statusCh)

	tempPath := <-statusCh
	if strings.Contains(tempPath, "Error") {
		return c.NoContent(http.StatusInternalServerError)
	} else {
		c.Response().WriteHeader(http.StatusOK)
		if _, err := c.Response().Write([]byte(tempPath + "\n")); err != nil {
			return err
		}

		c.Response().Flush()
	}

	fileStoragePath := <-statusCh
	if strings.Contains(fileStoragePath, "Error") {
		if _, err := c.Response().Write([]byte(fileStoragePath + "\n")); err != nil {
			return err
		}

		return nil
	}

	if err := h.MessengerService.SendFile(c.Request().Context(), h.UsersService, userLogin, req.Receiver, req.FileName, fileStoragePath); err != nil {
		go h.FilesService.DelFile(context.Background(), fileStoragePath)
		c.Logger().Errorf("Send file ERR: %s", err)
		resp := err.Error()
		if key != "" {
			resp, _ = crypto.StringEncrypt([]byte(resp), key)
		}
		c.Response().Write([]byte(resp))
		return nil
	}

	c.Response().Write([]byte("Success"))
	return nil
}

func (h *Handler) DelFile(c echo.Context, requestBody []byte, key string, userLogin string) error {
	var req DelFileRequest
	if err := json.Unmarshal(requestBody, &req); err != nil {
		h.Blocker.RegisterStrike(c.RealIP())
		return c.JSON(http.StatusBadRequest, "Invalid JSON")
	}

	if req.Filename == "" {
		h.Blocker.RegisterStrike(c.RealIP())
		return c.String(http.StatusBadRequest, "Did not receive all server data")
	}

	splittedFilename := strings.Split(req.Filename, ".")
	if len(splittedFilename) == 2 {
		if splittedFilename[1] != userLogin {
			resp := "Not access"
			if key != "" {
				resp, _ = crypto.StringEncrypt([]byte(resp), key)
			}
			return c.String(http.StatusBadRequest, resp)
		}
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

func (h *Handler) SetAvatar(c echo.Context, key string, userLogin string) error {
	statusCh := make(chan string)
	go h.FilesService.SaveAvatarTempPath(c.Request().Context(), key, statusCh)

	tempPath := <-statusCh
	if strings.Contains(tempPath, "Error") {
		return c.NoContent(http.StatusInternalServerError)
	} else {
		c.Response().WriteHeader(http.StatusOK)
		if _, err := c.Response().Write([]byte(tempPath + "\n")); err != nil {
			return err
		}

		c.Response().Flush()
	}

	storageAvatarPath := <-statusCh
	fmt.Println(storageAvatarPath)
	if strings.Contains(storageAvatarPath, "Error") {
		if _, err := c.Response().Write([]byte(storageAvatarPath + "\n")); err != nil {
			return err
		}

		return nil
	}

	if err := h.UsersService.SetAvatar(c.Request().Context(), userLogin, storageAvatarPath); err != nil {
		c.Logger().Error("Set avatar ERR %s", err)
		resp := err.Error()
		if key != "" {
			resp, _ = crypto.StringEncrypt([]byte(resp), key)
		}
		return c.String(http.StatusBadRequest, resp)
	}

	c.Logger().Infof("Success save avatar by %s", userLogin)

	c.Response().Write([]byte("Success"))
	return nil
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
	fmt.Println(storagePath)

	tempPath, err := h.FilesService.DownloadFileTempPath(c.Request().Context(), storagePath, key)
	if err != nil {
		c.Logger().Errorf("Generate temp path for Download ERR %s", err)
		resp := "Cant generate temp path"
		if key != "" {
			resp, _ = crypto.StringEncrypt([]byte(resp), key)
		}
		return c.String(http.StatusInternalServerError, resp)
	}

	if key != "" {
		tempPath, err = crypto.StringEncrypt([]byte(tempPath), key)
		if err != nil {
			c.Logger().Errorf("Encrypt temp path ERR %s", err)
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

	if strings.Split(req.StorageFilepath, ".")[1] != userLogin {
		resp := "Not access"
		if key != "" {
			resp, _ = crypto.StringEncrypt([]byte(resp), key)
		}
		return c.String(http.StatusInternalServerError, resp)
	}

	tempPath, err := h.FilesService.DownloadFileTempPath(c.Request().Context(), req.StorageFilepath, key)
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
