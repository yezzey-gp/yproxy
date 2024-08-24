package storage

import "github.com/yezzey-gp/yproxy/pkg/message"

func ResolveStorageSetting(settings []message.PutSettings, name, defaultVal string) string {

	for _, s := range settings {
		if s.Name == name {
			return s.Value
		}
	}

	return defaultVal
}
