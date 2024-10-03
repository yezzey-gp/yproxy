package config

type Vacuum struct {
	CheckBackup bool `json:"check_backup,default=true" toml:"check_backup,default=true" yaml:"check_backup,default=true"`
}
