package gits

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
)

func (r *Repo) Object(hash string) (*Object, error) {
	path := fmt.Sprintf("objects/%s/%s", hash[:2], hash[2:])

	content, err := r.fs.ReadFile(path)

	if err != nil {
		return nil, err
	}

	content, err = Zlib.Decompress(content)

	if err != nil {
		return nil, err
	}

	spaceIdx := bytes.IndexByte(content, ' ')

	if spaceIdx == -1 {
		return nil, fmt.Errorf("invalid object: no space found")
	}

	objectType := string(content[:spaceIdx])
	nullIdx := bytes.IndexByte(content[spaceIdx+1:], 0)

	if nullIdx == -1 {
		return nil, fmt.Errorf("invalid object: no null terminator found")
	}

	nullIdx += spaceIdx + 1

	sizeStr := string(content[spaceIdx+1 : nullIdx])
	size, err := strconv.Atoi(sizeStr)

	if err != nil {
		return nil, fmt.Errorf("invalid size: %w", err)
	}

	dataIndex := nullIdx + 1
	data := content[dataIndex:]

	if len(data) != size {
		return nil, fmt.Errorf("size mismatch: header says %d, got %d", size, len(data))
	}

	object := &Object{
		Hash: hash,
		Size: size,
		Data: data,
		Type: OBJ_TYPES_NUM[objectType],
	}

	if object.Type == 0 {
		return nil, fmt.Errorf("unknown object type: %s", objectType)
	}

	if object.Type == OBJ_COMMIT {
		kv := parseLinesKV(object.Data)

		if len(kv["tree"]) > 0 {
			object.TreeHash = kv["tree"][0]
		}

		object.ParentHashes = kv["parent"]
	}

	return object, nil
}

func (o *Object) Header() ([]byte, error) {
	if o.Type < 1 || o.Type > 4 {
		return nil, fmt.Errorf("invalid object type")
	}

	var buf bytes.Buffer

	// First byte: low 4 bits = size & 0x0F, bits 4-6 = type
	first := byte(o.Type<<4) | byte(o.Size&0x0F)
	o.Size >>= 4

	// Set continuation bit if more size bits follow
	if o.Size > 0 {
		first |= 0x80
	}

	buf.WriteByte(first)

	// Continuation bytes: 7 bits per byte, LSB chunk first
	for o.Size > 0 {
		b := byte(o.Size & 0x7F)
		o.Size >>= 7

		if o.Size > 0 {
			b |= 0x80 // more bytes follow
		}

		buf.WriteByte(b)
	}

	return buf.Bytes(), nil
}

func (o *Object) Tree() (map[string]uint8, error) {
	if o.Type != OBJ_TREE {
		return nil, fmt.Errorf("object is not a tree")
	}

	result := make(map[string]uint8)
	i := 0

	for i < len(o.Data) {
		// mode (ASCII until space)
		spaceIdx := i
		for spaceIdx < len(o.Data) && o.Data[spaceIdx] != ' ' {
			spaceIdx++
		}
		if spaceIdx >= len(o.Data) {
			return nil, fmt.Errorf("invalid format: mode not terminated")
		}

		// file name (until null byte)
		nullIdx := spaceIdx + 1
		for nullIdx < len(o.Data) && o.Data[nullIdx] != 0 {
			nullIdx++
		}
		if nullIdx >= len(o.Data) {
			return nil, fmt.Errorf("invalid format: filename not terminated")
		}

		// object id: 20 bytes binary SHA1
		hashStart := nullIdx + 1
		hashEnd := hashStart + 20
		if hashEnd > len(o.Data) {
			return nil, fmt.Errorf("invalid format: hash truncated")
		}

		hashHex := hex.EncodeToString(o.Data[hashStart:hashEnd])

		// Determine type from mode
		mode := string(o.Data[i:spaceIdx])

		var objType uint8

		switch {
		case strings.HasPrefix(mode, "40000"):
			objType = OBJ_TREE
		default:
			objType = OBJ_BLOB
		}

		result[hashHex] = objType

		// Move to next entry
		i = hashEnd
	}

	return result, nil
}
