package config

type Instance struct {
	StorageCnf Storage `json:"storage"`
	ProxyCnf   Proxy   `json:"proxy"`
}
