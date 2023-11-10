package config

type Storage struct {
	StorageEndpoint string `json:"storage_endpoint"`
	AccessKeyId     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
	StoragePrefix   string `json:"storage_prefix"`
	StorageBucket   string `json:"bucket"`
	GPGKeyId        string `json:"gpg_key_id"`
	GPGKeyPath      string `json:"gpg_key_path"`
}
