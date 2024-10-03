package proc

import (
	"fmt"
	"path"
	"strings"

	"github.com/pkg/errors"
	"github.com/yezzey-gp/yproxy/config"
	"github.com/yezzey-gp/yproxy/pkg/backups"
	"github.com/yezzey-gp/yproxy/pkg/database"
	"github.com/yezzey-gp/yproxy/pkg/message"
	"github.com/yezzey-gp/yproxy/pkg/storage"
	"github.com/yezzey-gp/yproxy/pkg/ylogger"
)

//go:generate mockgen -destination=../../../test/mocks/mock_object.go -package mocks -build_flags -mod=readonly github.com/wal-g/wal-g/pkg/storages/storage Object
type DeleteHandler interface {
	HandleDeleteGarbage(message.DeleteMessage) error
	HandleDeleteFile(message.DeleteMessage) error
}

type BasicDeleteHandler struct {
	BackupInterractor  backups.BackupInterractor
	DbInterractor      database.DatabaseInterractor
	StorageInterractor storage.StorageInteractor

	Cnf *config.Vacuum
}

func (dh *BasicDeleteHandler) HandleDeleteGarbage(msg message.DeleteMessage) error {
	fileList, err := dh.ListGarbageFiles(msg)
	if err != nil {
		return errors.Wrap(err, "failed to delete file")
	}

	if !msg.Confirm { //do not delete files if no confirmation flag provided
		return nil
	}

	var failed []string
	retryCount := 0
	for len(fileList) > 0 && retryCount < 10 {
		retryCount++
		for i := 0; i < len(fileList); i++ {

			if msg.CrazyDrop {
				ylogger.Zero.Debug().Str("path", fileList[i]).Msg("simply delete without any 'plan B'")
				err = dh.StorageInterractor.DeleteObject(fileList[i])

			} else {

				filePathParts := strings.Split(fileList[i], "/")

				destPath := path.Join(
					"trash",
					"segments_005",
					fmt.Sprintf("seg%d", msg.Segnum),
					"basebackups_005",
					"yezzey", filePathParts[len(filePathParts)-1])

				err = dh.StorageInterractor.MoveObject(fileList[i], destPath)
			}
			if err != nil {
				ylogger.Zero.Warn().AnErr("err", err).Str("file", fileList[i]).Msg("failed to obsolete file")
				failed = append(failed, fileList[i])
			}
		}
		fileList = failed
		failed = make([]string, 0)
	}

	if len(fileList) > 0 {
		ylogger.Zero.Error().Int("failed files count", len(fileList)).Msg("some files were not moved")
		ylogger.Zero.Error().Any("failed files", fileList).Msg("failed to move some files")
		return errors.Wrap(err, "failed to move some files")
	}

	return nil
}

func (dh *BasicDeleteHandler) HandleDeleteFile(msg message.DeleteMessage) error {
	err := dh.StorageInterractor.DeleteObject(msg.Name)
	if err != nil {
		ylogger.Zero.Error().AnErr("err", err).Msg("failed to delete file")
		return errors.Wrap(err, "failed to delete file")
	}
	return nil
}

func (dh *BasicDeleteHandler) ListGarbageFiles(msg message.DeleteMessage) ([]string, error) {
	//get firsr backup lsn
	var firstBackupLSN uint64
	var err error

	if dh.Cnf.CheckBackup {
		firstBackupLSN, err = dh.BackupInterractor.GetFirstLSN(msg.Segnum)
		if err != nil {
			ylogger.Zero.Error().AnErr("err", err).Msg("failed to get first lsn") //return or just assume there are no backups?
		}
		ylogger.Zero.Info().Uint64("lsn", firstBackupLSN).Msg("first backup LSN")
	} else {
		firstBackupLSN = ^uint64(0)
		ylogger.Zero.Info().Uint64("lsn", firstBackupLSN).Msg("omit first backup LSN")
	}

	//list files in storage
	ylogger.Zero.Info().Str("path", msg.Name).Msg("going to list path")
	objectMetas, err := dh.StorageInterractor.ListPath(msg.Name)
	if err != nil {
		return nil, errors.Wrap(err, "could not list objects")
	}
	ylogger.Zero.Info().Int("amount", len(objectMetas)).Msg("objects count")

	vi, ei, err := dh.DbInterractor.GetVirtualExpireIndexes(msg.Port)
	if err != nil {
		ylogger.Zero.Error().AnErr("err", err).Msg("failed to get indexes")
		return nil, errors.Wrap(err, "could not get virtual and expire indexes")
	}
	ylogger.Zero.Info().Msg("recieved virtual index and expire index")
	ylogger.Zero.Debug().Int("virtual", len(vi)).Msg("virtual index match count")
	ylogger.Zero.Debug().Int("expire", len(ei)).Msg("exprire index match count")

	filesToDelete := make([]string, 0)
	for i := 0; i < len(objectMetas); i++ {
		reworkedName := ReworkFileName(objectMetas[i].Path)
		ylogger.Zero.Debug().Str("reworked name", reworkedName).Msg("lookup chunk")

		if vi[reworkedName] {
			continue
		}

		lsn, ok := ei[reworkedName]
		ylogger.Zero.Debug().Uint64("lsn", lsn).Uint64("backup lsn", firstBackupLSN).Str("path", objectMetas[i].Path).Msg("comparing lsn")
		if lsn < firstBackupLSN || !ok {
			ylogger.Zero.Debug().Str("file", objectMetas[i].Path).
				Bool("file in expire index", ok).
				Bool("lsn is less than in first backup", lsn < firstBackupLSN).
				Msg("file does not persisnt in virtual index, nor needed for PITR, so will be deleted")
			filesToDelete = append(filesToDelete, objectMetas[i].Path)
		}
	}

	ylogger.Zero.Info().Int("amount", len(filesToDelete)).Msg("files will be deleted")

	return filesToDelete, nil
}

func ReworkFileName(str string) string {
	p1 := strings.Split(str, "/")
	p2 := p1[len(p1)-1]
	p3 := strings.Split(p2, "_")
	if len(p3) >= 4 {
		p2 = fmt.Sprintf("%s_%s_%s_%s_", p3[0], p3[1], p3[2], p3[3])
	}
	return p2
}
