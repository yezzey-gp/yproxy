package crypt

import (
	"bytes"
	"io"
	"os"

	"github.com/pkg/errors"
	"github.com/yezzey-gp/yproxy/config"
	"github.com/yezzey-gp/yproxy/pkg/ylogger"

	"github.com/ProtonMail/go-crypto/openpgp"
)

type Crypter interface {
	Decrypt(reader io.Reader) (io.Reader, error)
	Encrypt(writer io.WriteCloser) (io.WriteCloser, error)
}

type GPGCrypter struct {
	EntityList openpgp.EntityList

	cnf *config.Crypto
}

func NewCrypto(cnf *config.Crypto) Crypter {
	return &GPGCrypter{
		cnf: cnf,
	}
}

func (g *GPGCrypter) readKey(path string) (io.Reader, error) {
	byteData, err := os.ReadFile(path)

	if err != nil {
		return nil, err
	}

	return bytes.NewReader(byteData), nil
}

func (g *GPGCrypter) readPGPKey() (openpgp.EntityList, error) {
	gpgKeyReader, err := g.readKey(g.cnf.GPGKeyPath)

	if err != nil {
		return nil, err
	}

	entityList, err := openpgp.ReadArmoredKeyRing(gpgKeyReader)

	if err != nil {
		return nil, err
	}

	return entityList, nil
}

func (g *GPGCrypter) loadSecret() error {
	entityList, err := g.readPGPKey()

	if err != nil {
		return errors.WithStack(err)
	}

	g.EntityList = entityList

	return nil
}

func (g *GPGCrypter) Decrypt(reader io.Reader) (io.Reader, error) {
	err := g.loadSecret()
	if err != nil {
		return nil, err
	}
	ylogger.Zero.Debug().Str("gpg path", g.cnf.GPGKeyPath).Msg("loaded gpg key")

	md, err := openpgp.ReadMessage(reader, g.EntityList, nil, nil)

	if err != nil {
		return nil, errors.WithStack(err)
	}

	return md.UnverifiedBody, nil
}

func (g *GPGCrypter) Encrypt(writer io.WriteCloser) (io.WriteCloser, error) {
	err := g.loadSecret()
	if err != nil {
		return nil, err
	}
	ylogger.Zero.Debug().Str("gpg path", g.cnf.GPGKeyPath).Msg("loaded gpg key")

	encryptedWriter, err := openpgp.Encrypt(writer, g.EntityList, nil, nil, nil)

	if err != nil {
		return nil, errors.WithStack(err)
	}

	return encryptedWriter, nil
}
