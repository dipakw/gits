package gits

import (
	"bytes"
	"fmt"
)

func (repo *Repo) applyDelta(base string, reader *bytes.Reader) (uint8, []byte, error) {
	object, err := repo.Object(base)

	if err != nil {
		return 0, nil, err
	}

	baseSize, err := readSize(reader)

	if err != nil {
		return 0, nil, err
	}

	if baseSize != uint64(len(object.Data)) {
		return 0, nil, fmt.Errorf("base size mismatch: %d != %d", baseSize, len(object.Data))
	}

	resultSize, err := readSize(reader)

	if err != nil {
		return 0, nil, err
	}

	ops, err := parseDeltaOps(reader)

	if err != nil {
		return 0, nil, err
	}

	var buffer bytes.Buffer

	for _, op := range ops {
		// Copy.
		if op.Copy {
			buffer.Write(object.Data[op.Offset : op.Offset+op.Size])
		}

		// Insert.
		if !op.Copy {
			// fmt.Println("Insert", string(op.Data))
			buffer.Write(op.Data)
		}
	}

	if buffer.Len() != int(resultSize) {
		return 0, nil, fmt.Errorf("result size mismatch: %d != %d", buffer.Len(), resultSize)
	}

	return object.Type, buffer.Bytes(), nil
}
