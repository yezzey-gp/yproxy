package config

type Instance struct {
	StorageCnf Storage `json:"storage"`
	ProxyCnf   Proxy   `json:"proxy"`

	LogPath string `json:"log_path"`
}

var cfgInstance Instance

func InstanceConfig() *Instance {
	return &cfgInstance
}
