package gits

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
)

func (repo *Repo) Unpack(br *bufio.Reader) error {
	sig := make([]byte, 4)

	if _, err := io.ReadFull(br, sig); err != nil {
		return err
	}

	if string(sig) != "PACK" {
		return fmt.Errorf("invalid pack signature: %q", sig)
	}

	var version uint32

	if err := binary.Read(br, binary.BigEndian, &version); err != nil {
		return err
	}

	var objCount uint32

	if err := binary.Read(br, binary.BigEndian, &objCount); err != nil {
		return err
	}

	for i := uint32(0); i < objCount; i++ {
		typ, size, err := getPackObjectHeader(br)

		if err != nil {
			return err
		}

		content, base, err := getPackObjectContent(br, typ, size)

		if err != nil {
			return err
		}

		if typ == OBJ_REF_DELTA {
			reader := bytes.NewReader(content)
			baseID := hex.EncodeToString(base)

			typ, content, err = repo.applyDelta(baseID, reader)

			if err != nil {
				return err
			}
		}

		if typ == OBJ_OFS_DELTA {
			return fmt.Errorf("ofs-delta is not implemented")
		}

		// Object types stored without processing.
		// 1. commit | OBJ_COMMIT
		// 2. tree   | OBJ_TREE
		// 3. blob   | OBJ_BLOB
		// 4. tag    | OBJ_TAG

		if OBJ_TYPES_STR[typ] == "" {
			return fmt.Errorf("unknown object type: %d", typ)
		}

		header := fmt.Sprintf("%s %d\x00", OBJ_TYPES_STR[typ], len(content))
		objdata := append([]byte(header), content...)
		hashBytes := sha1.Sum(objdata)
		hashHex := hex.EncodeToString(hashBytes[:])
		compressed, err := Zlib.Compress(objdata)

		if err != nil {
			return err
		}

		objpath := fmt.Sprintf("objects/%s/%s", hashHex[:2], hashHex[2:])

		if err := repo.fs.WriteFile(objpath, compressed); err != nil {
			return err
		}
	}

	return nil
}
