package message

import "bytes"

func GetCstring(b []byte) (string, uint64) {
	offset := uint64(0)
	buff := bytes.NewBufferString("")

	for i := 0; i < len(b); i++ {
		offset++
		if b[i] == 0 {
			break
		}
		buff.WriteByte(b[i])
	}

	return buff.String(), offset
}
