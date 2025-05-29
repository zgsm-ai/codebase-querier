package config

type Database struct {
	Driver      string
	DataSource  string
	AutoMigrate struct {
		Enable  bool
		Scripts string `json:",optional"`
	}
}
