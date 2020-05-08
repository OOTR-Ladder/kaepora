package util

import (
	"bytes"
	"compress/zlib"
	"database/sql/driver"
	"fmt"
	"io"
	"log"
)

// ZLIBBlob is an arbitrary blob stored as a zlib-compressed binary blob
// The data is compressed at rest.
type ZLIBBlob []byte

// NewZLIBBlob creates a new blob from uncompressed data.
func NewZLIBBlob(v []byte) (ZLIBBlob, error) {
	var b bytes.Buffer
	w, err := zlib.NewWriterLevel(&b, zlib.BestCompression)
	if err != nil {
		return nil, err
	}

	if _, err := w.Write(v); err != nil {
		return nil, err
	}

	if err := w.Close(); err != nil {
		return nil, err
	}

	return ZLIBBlob(b.Bytes()), nil
}

func (a ZLIBBlob) Value() (driver.Value, error) {
	return []byte(a), nil
}

func (a *ZLIBBlob) Scan(src interface{}) error {
	switch src := src.(type) {
	case []byte:
		var b bytes.Buffer
		b.Write(src)
		*a = ZLIBBlob(b.Bytes())
		return nil
	case string:
		var b bytes.Buffer
		b.WriteString(src)
		*a = ZLIBBlob(b.Bytes())
		return nil
	default:
		return fmt.Errorf("expected []byte or string, got %T", src)
	}
}

// Uncompressed returns a reader of uncompressed data.
func (a ZLIBBlob) Uncompressed() io.Reader {
	r, err := zlib.NewReader(bytes.NewReader([]byte(a)))
	if err != nil {
		log.Printf("error: invalid zlib blob of length %d", len([]byte(a)))
		return bytes.NewReader([]byte{})
	}

	return r
}
