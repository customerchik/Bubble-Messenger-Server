package handlers

import (
	"bubble/internal/cookie"
	"bubble/internal/crypto"
	"bubble/internal/crypto/tls"
	"io"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

func (rh *TLSHandler) Router(c echo.Context) error {
	var req tls.TLSData
	if err := c.Bind(&req); err != nil {
		rh.Blocker.RegisterStrike(c.RealIP())
		return c.JSON(http.StatusBadRequest, "Invalid JSON")
	}

	if req.Device == "" || req.Login == "" || req.Password == "" {
		rh.Blocker.RegisterStrike(c.RealIP())
		return c.String(http.StatusBadRequest, "Did not receive all server data")
	}

	key, _, err := rh.Auth.GetAuthInfo(c.Request().Context(), req.Login, req.Password, req.Device)
	if err != nil {
		c.Logger().Errorf("GetAuthInfo IP %s ERR: %s", err, c.RealIP())
		return c.String(http.StatusInternalServerError, err.Error())
	}

	rh.UsersUsedRequestsNonces.Lock()
	if err := tls.QueryCheck(rh.UsersUsedRequestsNonces.Nonces, &req, key); err != nil {
		switch err {
		case tls.DecryptErr:
			c.Logger().Warnf("%s. DeviceID %s", err, req.Device)
			encryptResp, _ := crypto.StringEncrypt([]byte(err.Error()), key)
			return c.String(http.StatusInternalServerError, encryptResp)
		case tls.ParseTimeErr, tls.ParseTimeErr:
			c.Logger().Warnf("%s. DeviceID %s, SendTime %s", err, req.Device, req.SendTime)
			encryptResp, _ := crypto.StringEncrypt([]byte(err.Error()), key)
			return c.String(http.StatusBadRequest, encryptResp)
		case tls.NonceUseTwice:
			c.Logger().Warnf("%s. DeviceID %s, Nonce %s, SendTime %s", req.Device, req.Nonce, req.SendTime)
			encryptResp, _ := crypto.StringEncrypt([]byte(err.Error()), key)
			return c.String(http.StatusBadRequest, encryptResp)
		}
	}
	rh.UsersUsedRequestsNonces.Unlock()

	req.RequestBody, err = crypto.StringDecryptByte(req.RequestBody, key)
	if err != nil {
		c.Logger().Warnf("Decrypt error. DeviceID %s", req.Device)
		encryptResp, _ := crypto.StringEncrypt([]byte("Decrypted error"), key)
		return c.String(http.StatusInternalServerError, encryptResp)
	}
	req.Login, err = crypto.StringDecrypt(req.Login, key)
	if err != nil {
		c.Logger().Warnf("Decrypt error. DeviceID %s", req.Device)
		encryptResp, _ := crypto.StringEncrypt([]byte("Decrypted error"), key)
		return c.String(http.StatusInternalServerError, encryptResp)
	}

	URLPath := strings.TrimPrefix(c.Request().URL.Path, "/")
	URLPath, err = crypto.StringDecrypt(URLPath, key)
	if err != nil {
		c.Logger().Warnf("Decrypt error. DeviceID %s", req.Device)
		encryptResp, _ := crypto.StringEncrypt([]byte("Decrypted error"), key)
		return c.String(http.StatusInternalServerError, encryptResp)
	}

	switch URLPath {
	case "searchuser":
		return rh.Handlers.SearchUser(c, req.RequestBody, key)
	case "sendmessage":
		return rh.Handlers.SendMessage(c, req.RequestBody, key, req.Login)
	case "checkmessage":
		return rh.Handlers.CheckMessage(c, key, req.Login)
	case "delmessages":
		return rh.Handlers.DelMessages(c, key, req.Login)
	case "setavatar":
		return rh.Handlers.SetAvatar(c, key, req.Login)
	case "getavatar":
		return rh.Handlers.GetAvatar(c, req.RequestBody, key)
	case "sendfile":
		return rh.Handlers.SendFile(c, req.RequestBody, key, req.Login)
	case "delfile":
		return rh.Handlers.DelFile(c, req.RequestBody, key, req.Login)
	case "getfile":
		return rh.Handlers.GetFile(c, req.RequestBody, key, req.Login)
	default:
		return c.NoContent(http.StatusNotFound)
	}
}

func (rh *TLSHandler) RouterSendFile(c echo.Context) error {
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return err
	}

	login, err := c.Cookie(cookie.UserLogin)
	if err != nil {
		return c.String(http.StatusBadRequest, "Cookie mycookie is missing")
	}

	return rh.Handlers.SendFile(c, body, "", login.Value)
}

func (rh *TLSHandler) RouterDelFile(c echo.Context) error {
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return err
	}

	login, err := c.Cookie(cookie.UserLogin)
	if err != nil {
		return c.String(http.StatusBadRequest, "Cookie mycookie is missing")
	}

	return rh.Handlers.DelFile(c, body, "", login.Value)
}

func (rh *TLSHandler) RouterSetAvatar(c echo.Context) error {
	login, err := c.Cookie(cookie.UserLogin)
	if err != nil {
		return c.String(http.StatusBadRequest, "Cookie mycookie is missing")
	}

	return rh.Handlers.SetAvatar(c, "", login.Value)
}

func (rh *TLSHandler) RouterGetAvatar(c echo.Context) error {
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return err
	}

	return rh.Handlers.GetAvatar(c, body, "")
}

func (rh *TLSHandler) RouterSearchUser(c echo.Context) error {
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return err
	}

	return rh.Handlers.SearchUser(c, body, "")
}

func (rh *TLSHandler) RouterSendMessage(c echo.Context) error {
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return err
	}

	login, err := c.Cookie(cookie.UserLogin)
	if err != nil {
		return c.String(http.StatusBadRequest, "Cookie mycookie is missing")
	}

	return rh.Handlers.SendMessage(c, body, "", login.Value)
}

func (rh *TLSHandler) RouterCheckMessages(c echo.Context) error {
	login, err := c.Cookie(cookie.UserLogin)
	if err != nil {
		return c.String(http.StatusBadRequest, "Cookie mycookie is missing")
	}

	return rh.Handlers.CheckMessage(c, "", login.Value)
}

func (rh *TLSHandler) RouterDelMessages(c echo.Context) error {
	login, err := c.Cookie(cookie.UserLogin)
	if err != nil {
		return c.String(http.StatusBadRequest, "Cookie mycookie is missing")
	}

	return rh.Handlers.DelMessages(c, "", login.Value)
}

func (rh *TLSHandler) RouterGetFile(c echo.Context) error {
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return err
	}

	login, err := c.Cookie(cookie.UserLogin)
	if err != nil {
		return c.String(http.StatusBadRequest, "Cookie mycookie is missing")
	}

	return rh.Handlers.GetFile(c, body, "", login.Value)
}
