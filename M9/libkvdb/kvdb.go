package kvdb

import (
	"encoding/binary"
	"errors"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"sync"
	"syscall"
)

const (
	magicNumber       = 0x4B564442 // "KVDB"
	version           = 1
	headerSize  int64 = 8 // magic(4) + version(4)
)

// KVDB represents a key-value database
type KVDB struct {
	path  string
	file  *os.File
	index map[string]int64
	mu    sync.RWMutex
}

type record struct {
	kind     byte
	keyLen   uint32
	valLen   uint32
	checkSum uint32
	key      []byte
	value    []byte
}

func (db *KVDB) recover() error {
	stat, err := db.file.Stat()
	if err != nil {
		return err
	}
	if stat.Size() == 0 {
		return db.writeHeader()
	}
	offset := headerSize
	for {
		rec, nextOffset, err := db.readRecord(offset)
		if err == io.EOF {
			break
		}
		if err != nil {
			return db.file.Truncate(offset)
		}
		db.index[string(rec.key)] = offset
		offset = nextOffset
	}
	return nil
}

func (db *KVDB) writeHeader() error {
	header := make([]byte, headerSize)
	binary.BigEndian.PutUint32(header[0:4], magicNumber)
	binary.BigEndian.PutUint32(header[4:8], version)
	_, err := db.file.Write(header)
	return err
}
func (db *KVDB) writeRecord(rec *record) error {
	checkSum, err := db.calculateChecksum(rec)
	if err != nil {
		return err
	}
	rec.checkSum = checkSum
	stat, err := db.file.Stat()
	if err != nil {
		return err
	}

	size := stat.Size()
	buf := db.serializeRecord(rec)
	// recover ReadAt读取时，不移动文件指针，offset: 0，覆盖文件头
	//_, err = db.file.Write(buf)
	_, err = db.file.WriteAt(buf, size)
	if err != nil {
		return err
	}

	db.index[string(rec.key)] = size
	return nil
}

func (db *KVDB) readRecord(offset int64) (*record, int64, error) {
	header := make([]byte, 13)
	_, err := db.file.ReadAt(header, offset)
	if err != nil {
		return nil, 0, err
	}
	rec := &record{
		kind:     header[0],
		keyLen:   binary.BigEndian.Uint32(header[1:5]),
		valLen:   binary.BigEndian.Uint32(header[5:9]),
		checkSum: binary.BigEndian.Uint32(header[9:13]),
	}

	dataLen := rec.keyLen + rec.valLen
	data := make([]byte, dataLen)
	_, err = db.file.ReadAt(data, offset+13)
	if err != nil {
		return nil, 0, err
	}
	rec.key = data[:rec.keyLen]
	rec.value = data[rec.keyLen:dataLen]

	expectedSum, err := db.calculateChecksum(rec)
	if err != nil || rec.checkSum != expectedSum {
		return nil, 0, errors.New("checksum mismatch")
	}
	return rec, offset + int64(13+dataLen), nil
}

func (db *KVDB) serializeRecord(rec *record) []byte {
	size := 13 + rec.keyLen + rec.valLen
	buf := make([]byte, size)
	buf[0] = rec.kind
	binary.BigEndian.PutUint32(buf[1:5], rec.keyLen)
	binary.BigEndian.PutUint32(buf[5:9], rec.valLen)
	binary.BigEndian.PutUint32(buf[9:13], rec.checkSum)
	copy(buf[13:], rec.key)
	copy(buf[13+rec.keyLen:], rec.value)
	return buf
}

func (db *KVDB) calculateChecksum(rec *record) (uint32, error) {
	h := crc32.NewIEEE()
	_, err := h.Write([]byte{rec.kind})
	if err != nil {
		return 0, err
	}
	err = binary.Write(h, binary.LittleEndian, rec.keyLen)
	if err != nil {
		return 0, err
	}
	err = binary.Write(h, binary.LittleEndian, rec.valLen)
	if err != nil {
		return 0, err
	}
	_, err = h.Write(rec.key)
	if err != nil {
		return 0, err
	}
	_, err = h.Write(rec.value)
	if err != nil {
		return 0, err
	}
	return h.Sum32(), nil
}

// Open initializes the database and associates it with the given path.
// If the path doesn't exist, creates an empty database.
// Returns error if operation fails.
func Open(path string) (*KVDB, error) {
	// - Create file if not exists
	dir := filepath.Dir(path)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return nil, err
	}
	// - Open file with appropriate flags
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}
	err = syscall.Flock(int(file.Fd()), syscall.LOCK_EX)
	if err != nil {
		err = file.Close()
		if err != nil {
			return nil, err
		}
		return nil, err
	}
	// - Initialize database structure
	db := &KVDB{
		path:  path,
		file:  file,
		index: make(map[string]int64),
		mu:    sync.RWMutex{},
	}
	// - Handle crash recovery if needed
	err = db.recover()
	if err != nil {
		return nil, err
	}

	return db, nil
}

// Put sets the value for the given key.
// If key already exists, overwrites the previous value.
// Returns error if operation fails.
func (db *KVDB) Put(key, value string) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	// - Write key-value pair to database
	rec := &record{
		kind:   0x00,
		keyLen: uint32(len(key)),
		valLen: uint32(len(value)),
		key:    []byte(key),
		value:  []byte(value),
	}
	err := db.writeRecord(rec)
	if err != nil {
		return err
	}
	// - Ensure crash consistency (use fsync/fdatasync)
	err = db.file.Sync()
	if err != nil {
		return err
	}
	// - Handle concurrent access (file locking)
	return nil
}

// Get retrieves the value for the given key.
// Returns the value and nil error if found.
// Returns empty string and error if key doesn't exist.
func (db *KVDB) Get(key string) (string, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	// - Search for key in database
	offset, ok := db.index[key]
	if !ok {
		return "", errors.New("key not found")
	}
	rec, _, err := db.readRecord(offset)
	if err != nil {
		return "", err
	}
	return string(rec.value), nil
	// - Return corresponding value
	// - Handle concurrent access
}

// Close closes the database and releases associated resources.
// Returns error if operation fails.
func (db *KVDB) Close() error {
	// - Flush any pending writes
	err := db.file.Sync()
	if err != nil {
		return err
	}
	// 不能对已关闭的文件解锁
	// - Release locks (must be done before closing file)
	err = syscall.Flock(int(db.file.Fd()), syscall.LOCK_UN)
	if err != nil {
		return err
	}
	// - Close file descriptor
	err = db.file.Close()
	if err != nil {
		return err
	}
	return nil
}
