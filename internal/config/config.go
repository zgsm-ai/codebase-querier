package config

import (
	"errors"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest"
)

type Config struct {
	rest.RestConf

	Auth struct {
		UserInfoHeader string
	}
	DBPrimary struct {
		Driver     string `json:"Driver"`
		DataSource string `json:"DataSource"`
	} `json:"DBPrimary"`

	DBReplicas []struct {
		Driver     string `json:"Driver"`
		DataSource string `json:"DataSource"`
	} `json:"DBReplicas"`

	RedisConf redis.RedisConf
}

// Validate 实现 Validator 接口
func (c Config) Validate() error {
	if len(c.Name) == 0 {
		return errors.New("name 不能为空")
	}
	return nil
}
