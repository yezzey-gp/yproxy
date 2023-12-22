package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v2"
)

type Instance struct {
	StorageCnf Storage `json:"storage" toml:"storage" yaml:"storage"`
	ProxyCnf   Proxy   `json:"proxy" toml:"proxy" yaml:"proxy"`

	CryptoCnf Crypto `json:"crypto" toml:"crypto" yaml:"crypto"`

	LogPath    string `json:"log_path" toml:"log_path" yaml:"log_path"`
	LogLevel   string `json:"log_level" toml:"log_level" yaml:"log_level"`
	SocketPath string `json:"socket_path" toml:"socket_path" yaml:"socket_path"`

	SystemdNotificationsDebug bool `json:"sd_notifications_debug" toml:"sd_notifications_debug" yaml:"sd_notifications_debug"`
	systemdSocketPath         string
}

func (i *Instance) ReadSystemdSocketPath() {
	path := os.Getenv("NOTIFY_SOCKET")
	if path != "" {
		i.systemdSocketPath = path
	}
}

func (i *Instance) GetSystemdSocketPath() string {
	return i.systemdSocketPath
}

var cfgInstance Instance

func InstanceConfig() *Instance {
	return &cfgInstance
}

func initInstanceConfig(file *os.File, cfgInstance *Instance) error {
	if strings.HasSuffix(file.Name(), ".toml") {
		_, err := toml.NewDecoder(file).Decode(cfgInstance)
		return err
	}
	if strings.HasSuffix(file.Name(), ".yaml") {
		return yaml.NewDecoder(file).Decode(&cfgInstance)
	}
	if strings.HasSuffix(file.Name(), ".json") {
		return json.NewDecoder(file).Decode(&cfgInstance)
	}
	return fmt.Errorf("unknown config format type: %s. Use .toml, .yaml or .json suffix in filename", file.Name())
}

const (
	DefaultStorageConcurrency = 100
)

func EmbedDefaults(cfgInstance *Instance) {
	if cfgInstance.StorageCnf.StorageConcurrency == 0 {
		cfgInstance.StorageCnf.StorageConcurrency = DefaultStorageConcurrency
	}
}

func LoadInstanceConfig(cfgPath string) error {
	var cfg Instance
	file, err := os.Open(cfgPath)
	if err != nil {
		cfgInstance = cfg
		return err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Fatalf("failed to close config file: %v", err)
		}
	}(file)

	if err := initInstanceConfig(file, &cfg); err != nil {
		cfgInstance = cfg
		return err
	}

	configBytes, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		cfgInstance = cfg
		return err
	}

	cfg.ReadSystemdSocketPath()
	EmbedDefaults(&cfg)

	log.Println("Running config:", string(configBytes))
	cfgInstance = cfg
	return nil
}
