package config

// CodeBaseStoreConf 代码库存储配置
type CodeBaseStoreConf struct {
	// 具体存储类型的配置，使用内嵌结构实现
	Local LocalStoreConf `json:",optional"`
	Minio MinioStoreConf `json:",optional"`
}

// LocalStoreConf 本地文件系统存储配置
type LocalStoreConf struct {
	// 本地存储路径
	BasePath string
}

// MinioStoreConf MinIO对象存储配置
type MinioStoreConf struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	UseSSL          bool
	Bucket          string
}
