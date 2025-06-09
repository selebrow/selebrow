package controllers

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/selebrow/selebrow/pkg/dto"
)

type ConfigController struct {
	data map[string]string
}

func NewConfigController(data map[string]string) *ConfigController {
	return &ConfigController{
		data: data,
	}
}

func (cc *ConfigController) List(c echo.Context) error {
	cfg := &dto.Config{
		Files: make(map[string]dto.ConfigFile),
	}
	for n, d := range cc.data {
		sum := sha256.Sum256([]byte(d))
		cfg.Files[n] = dto.ConfigFile{
			SHA256Sum: hex.EncodeToString(sum[:]),
		}
	}
	return c.JSON(http.StatusOK, cfg)
}

func (cc *ConfigController) GetConfig(c echo.Context) error {
	name := c.Param("name")
	if d, ok := cc.data[name]; ok {
		return c.String(http.StatusOK, d)
	}
	return c.NoContent(http.StatusNotFound)
}
