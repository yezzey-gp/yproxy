package config

type Storage struct {
	StorageEndpoint string `json:"storage_endpoint" toml:"storage_endpoint" yaml:"storage_endpoint"`
	AccessKeyId     string `json:"access_key_id" toml:"access_key_id" yaml:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key" toml:"secret_access_key" yaml:"secret_access_key"`
	StoragePrefix   string `json:"storage_prefix" toml:"storage_prefix" yaml:"storage_prefix"`
	StorageBucket   string `json:"bucket" toml:"bucket" yaml:"bucket"`
	GPGKeyId        string `json:"gpg_key_id" toml:"gpg_key_id" yaml:"gpg_key_id"`
	GPGKeyPath      string `json:"gpg_key_path" toml:"gpg_key_path" yaml:"gpg_key_path"`
}
