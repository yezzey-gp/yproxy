package message

import (
	"bytes"
	"encoding/binary"
	"github.com/yezzey-gp/yproxy/pkg/storage"
)

type ObjectMetaMessage struct {
	Content []*storage.S3ObjectMeta
}

var _ ProtoMessage = &ObjectMetaMessage{}

func NewObjectMetaMessage(content []*storage.S3ObjectMeta) *ObjectMetaMessage {
	return &ObjectMetaMessage{
		Content: content,
	}
}

func (c *ObjectMetaMessage) Encode() []byte {
	bt := []byte{
		byte(MessageTypeObjectMeta),
		0,
		0,
		0,
	}

	for _, objMeta := range c.Content {
		bt = append(bt, []byte(objMeta.Path)...)
		bt = append(bt, 0)

		bn := make([]byte, 8)
		binary.BigEndian.PutUint64(bn, uint64(objMeta.Size))
		bt = append(bt, bn...)
	}

	ln := len(bt) + 8
	bs := make([]byte, 8)
	binary.BigEndian.PutUint64(bs, uint64(ln))
	return append(bs, bt...)
}

func (c *ObjectMetaMessage) Decode(body []byte) {
	objMetas := make([]*storage.S3ObjectMeta, 0)
	for len(body) > 0 {
		name, index := c.GetString(body)
		size := int64(binary.BigEndian.Uint64(body[index : index+8]))

		objMetas = append(objMetas, &storage.S3ObjectMeta{
			Path: name,
			Size: size,
		})
		body = body[index+8:]
	}
}

func (c *ObjectMetaMessage) GetString(b []byte) (string, int) {
	buff := bytes.NewBufferString("")

	i := 0
	for ; i < len(b); i++ {
		if b[i] == 0 {
			break
		}
		buff.WriteByte(b[i])
	}

	return buff.String(), i + 1
}
