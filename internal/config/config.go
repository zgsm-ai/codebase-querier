package config

import (
	"errors"
	"github.com/zeromicro/go-zero/rest"
)

type Config struct {
	rest.RestConf
	Auth struct {
		UserInfoHeader string
	}
	DB struct {
		Driver     string
		DataSource string
	}
	CodeBaseStore CodeBaseStoreConf
	MessageQueue  MessageQueueConf
	IndexJob      IndexJob
	VectorStore   VectorStoreConf
}

// Validate 实现 Validator 接口
func (c Config) Validate() error {
	if len(c.Name) == 0 {
		return errors.New("name 不能为空")
	}
	return nil
}
