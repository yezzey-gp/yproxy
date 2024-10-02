package backups

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/yezzey-gp/yproxy/pkg/ylogger"
)

type BackupLSN struct {
	Lsn uint64 `json:"LSN"`
}

//go:generate mockgen -destination=pkg/mock/backups.go -package=mock
type BackupInterractor interface {
	GetFirstLSN(seg uint64) (uint64, error)
}

type WalgBackupInterractor struct { //TODO: rewrite to using s3 instead of wal-g cmd
}

// get lsn of the oldest backup
func (b *WalgBackupInterractor) GetFirstLSN(seg uint64) (uint64, error) {
	cmd := exec.Command("/usr/bin/wal-g", "st", "ls", fmt.Sprintf("segments_005/seg%d/basebackups_005/", seg), "--config=/etc/wal-g/wal-g.yaml")
	ylogger.Zero.Debug().Any("flags", cmd.Args).Msg("Command args")
	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		ylogger.Zero.Debug().AnErr("error", err).Msg("Failed to run st ls")
		return 0, err
	}
	lines := strings.Split(out.String(), "\n")

	minLSN := BackupLSN{Lsn: ^uint64(0)}
	for _, line := range lines {
		if !strings.Contains(line, ".json") {
			continue
		}
		parts := strings.Split(line, " ")
		fileName := parts[len(parts)-1]

		ylogger.Zero.Debug().Str("file: %s", fmt.Sprintf("segments_005/seg%d/basebackups_005/%s", seg, fileName)).Msg("check lsn in file")
		catCmd := exec.Command("/usr/bin/wal-g", "st", "cat", fmt.Sprintf("segments_005/seg%d/basebackups_005/%s", seg, fileName), "--config=/etc/wal-g/wal-g.yaml")

		var catOut bytes.Buffer
		catCmd.Stdout = &catOut

		err = catCmd.Run()
		if err != nil {
			ylogger.Zero.Debug().AnErr("error", err).Msg("Failed to run st cat")
			return 0, err
		}
		lsn := BackupLSN{}
		err = json.Unmarshal(catOut.Bytes(), &lsn)

		if lsn.Lsn < minLSN.Lsn {
			minLSN.Lsn = lsn.Lsn
		}
	}

	return minLSN.Lsn, err
}
