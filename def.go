package gits

const (
	OBJ_COMMIT    = 1
	OBJ_TREE      = 2
	OBJ_BLOB      = 3
	OBJ_TAG       = 4
	OBJ_OFS_DELTA = 6
	OBJ_REF_DELTA = 7
)

var OBJ_TYPES_NUM = map[string]uint8{
	"commit":    OBJ_COMMIT,
	"tree":      OBJ_TREE,
	"blob":      OBJ_BLOB,
	"tag":       OBJ_TAG,
	"ofs-delta": OBJ_OFS_DELTA,
	"ref-delta": OBJ_REF_DELTA,
}

var OBJ_TYPES_STR = map[uint8]string{
	OBJ_COMMIT:    "commit",
	OBJ_TREE:      "tree",
	OBJ_BLOB:      "blob",
	OBJ_TAG:       "tag",
	OBJ_OFS_DELTA: "ofs-delta",
	OBJ_REF_DELTA: "ref-delta",
}

var ADVERTISE_CAPS = []string{
	// "multi_ack",
	// "multi_ack_detailed",
	// "thin-pack",
	// "side-band",
	// "side-band-64k",
	// "ofs-delta",
	"report-status",
	"agent=gits/dev",
}

const (
	FS_TYPE_FILE = 1
	FS_TYPE_DIR  = 2
	FS_TYPE_ALL  = FS_TYPE_FILE | FS_TYPE_DIR
)

type Head struct {
	NoHead   bool   // If the head is not found.
	Detached bool   // Head contains hash.
	Unborn   bool   // The ref file is not found.
	Ref      string // E.g: refs/heads/main
	Hash     string // Content of the ref file.
}

type Config struct {
	Dir  string
	Name string
	FS   func(root string) (FS, error)
}

type Repo struct {
	conf *Config
	fs   FS
}

type Object struct {
	Hash         string
	Type         uint8
	Size         int
	TreeHash     string
	ParentHashes []string
	Data         []byte
}

type Negotiation struct {
	Wants map[string]bool
	Haves map[string]bool
	Caps  map[string]bool
	Agent string
	Done  bool
	EOF   bool
}

type DeltaOp struct {
	Copy   bool
	Offset uint64
	Size   uint64
	Data   []byte
}

// ///////// Interfaces ///////////
type FS interface {
	// Read a single file from the FS
	ReadFile(path string) ([]byte, error)

	// Write a single file to the FS
	WriteFile(path string, data []byte) error

	// [name]: [TYPE, SIZE] -> [TYPE: 1 = file, 2 = dir, SIZE: size in bytes]
	// If level is -1, scan all files and directories
	// It returns the files only like: Scan("refs", -1) will return all files in "refs/tags/v1.0.0", etc.
	Scan(path string, include uint8, level int) (map[string][]int, error)

	// [name]: [TYPE, SIZE] -> [TYPE: 0 = not found, 1 = file, 2 = dir, SIZE: size in bytes]
	Stat(path string) []int

	// Create a dir recursively.
	Mkdir(path string) error

	// Change dir.
	Cd(path string) error

	// Get present working directory.
	Pwd() string
}
