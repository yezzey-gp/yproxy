package config

type Proxy struct {
	ConsolePort string `json:"console_port" toml:"console+port" yaml:"console_port"`
}
