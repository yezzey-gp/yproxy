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
	GetFirstLSN(int) (uint64, error)
}

type WalgBackupInterractor struct {
}

// get lsn of the oldest backup
func (b *WalgBackupInterractor) GetFirstLSN(seg int) (uint64, error) {
	cmd := exec.Command("/usr/bin/wal-g", "st", "ls", fmt.Sprintf("segments_005/seg%d/basebackups_005/", seg), "--config=/etc/wal-g/wal-g.yaml")
	ylogger.Zero.Debug().Any("flags", cmd.Args).Msg("Command args")
	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		ylogger.Zero.Debug().AnErr("error", err).Msg("Failed to run st ls")
		return 0, err
	}
	p1 := strings.Split(out.String(), "\n")

	minLSN := BackupLSN{Lsn: ^uint64(0)}
	for _, line := range p1 {
		if !strings.Contains(line, ".json") {
			continue
		}
		p2 := strings.Split(line, " ")
		p3 := p2[len(p2)-1]

		ylogger.Zero.Debug().Str("file: %s", fmt.Sprintf("segments_005/seg%d/basebackups_005/%s", seg, p3)).Msg("check lsn in file")
		cmd2 := exec.Command("/usr/bin/wal-g", "st", "cat", fmt.Sprintf("segments_005/seg%d/basebackups_005/%s", seg, p3), "--config=/etc/wal-g/wal-g.yaml")

		var out2 bytes.Buffer
		cmd2.Stdout = &out2

		err = cmd2.Run()
		if err != nil {
			ylogger.Zero.Debug().AnErr("error", err).Msg("Failed to run st cat")
			return 0, err
		}
		lsn := BackupLSN{}
		err = json.Unmarshal(out2.Bytes(), &lsn)

		if lsn.Lsn < minLSN.Lsn {
			minLSN.Lsn = lsn.Lsn
		}
	}

	return minLSN.Lsn, err
}
