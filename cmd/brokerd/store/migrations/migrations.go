// Code generated by go-bindata. (@generated) DO NOT EDIT.

 //Package migrations generated by go-bindata.// sources:
// migrations/001_init.down.sql
// migrations/001_init.up.sql
// migrations/002_rw.down.sql
// migrations/002_rw.up.sql
// migrations/003_providers.down.sql
// migrations/003_providers.up.sql
// migrations/004_status_enums.down.sql
// migrations/004_status_enums.up.sql
// migrations/005_payload_size.down.sql
// migrations/005_payload_size.up.sql
// migrations/006_add_dealstartoffset.down.sql
// migrations/006_add_dealstartoffset.up.sql
package migrations

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func bindataRead(data []byte, name string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("read %q: %v", name, err)
	}

	var buf bytes.Buffer
	_, err = io.Copy(&buf, gz)
	clErr := gz.Close()

	if err != nil {
		return nil, fmt.Errorf("read %q: %v", name, err)
	}
	if clErr != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

type asset struct {
	bytes []byte
	info  os.FileInfo
}

type bindataFileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
}

// Name return file name
func (fi bindataFileInfo) Name() string {
	return fi.name
}

// Size return file size
func (fi bindataFileInfo) Size() int64 {
	return fi.size
}

// Mode return file mode
func (fi bindataFileInfo) Mode() os.FileMode {
	return fi.mode
}

// ModTime return file modify time
func (fi bindataFileInfo) ModTime() time.Time {
	return fi.modTime
}

// IsDir return file whether a directory
func (fi bindataFileInfo) IsDir() bool {
	return fi.mode&os.ModeDir != 0
}

// Sys return file is sys mode
func (fi bindataFileInfo) Sys() interface{} {
	return nil
}

var __001_initDownSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x72\x09\xf2\x0f\x50\x08\x71\x74\xf2\x71\x55\x28\xcd\x2b\xc8\xcc\x8b\xcf\xca\x4f\x2a\xb6\xe6\x42\x12\x4e\x49\x4d\xcc\x41\x15\x29\x2e\xc9\x2f\x4a\x4c\x4f\x8d\x2f\x4a\x2d\x2c\x4d\x2d\x2e\x41\x95\x4c\x4a\x2c\x49\xce\x48\x2d\xb6\xe6\x02\x04\x00\x00\xff\xff\xbc\x58\x40\x6e\x5a\x00\x00\x00")

func _001_initDownSqlBytes() ([]byte, error) {
	return bindataRead(
		__001_initDownSql,
		"001_init.down.sql",
	)
}

func _001_initDownSql() (*asset, error) {
	bytes, err := _001_initDownSqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "001_init.down.sql", size: 90, mode: os.FileMode(436), modTime: time.Unix(1628018891, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var __001_initUpSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xe4\x55\x41\x6f\xdb\x3c\x0c\xbd\xe7\x57\xf0\xd6\x06\xe8\xe1\xbb\xf7\xe4\x2f\x55\x06\x63\xa9\x53\x38\x2a\xd0\x9e\x04\xda\x62\x52\xad\xaa\xe5\xc9\x52\xd7\xec\xd7\x0f\xb1\xe3\x34\x8e\xad\xba\xcb\x65\x03\x76\xd5\x23\x29\xf2\x91\x7c\x9c\xa5\x2c\xe2\x0c\x78\xf4\xff\x82\x41\x3c\x87\x64\xc9\x81\x3d\xc4\x2b\xbe\x82\x0c\x5d\xfe\x44\x15\x5c\x4e\x00\x00\x94\x04\x47\x6f\x0e\xee\xd2\xf8\x36\x4a\x1f\xe1\x2b\x7b\xbc\xaa\x81\xca\xa1\xf3\x55\x03\xee\xbc\x93\xfb\xc5\xa2\x41\x2c\x95\x62\x8d\xb9\x33\x16\x54\x71\x0a\x4a\x42\x2d\xa4\xb7\xe8\x94\x29\x06\xf0\x12\xb7\xda\xa0\x14\x79\xfb\xf1\x09\xac\x28\xa7\x8f\xc1\x4a\xfd\x24\xc8\xd4\xa6\x1f\x3b\x47\x2b\xbc\xd5\x5d\x57\xb8\x61\xf3\xe8\x7e\xc1\xe1\xe2\xe2\xdd\x4a\x95\xeb\xaa\xff\x4b\xd8\x14\xa5\xb4\xd5\x88\xb1\x54\x15\x6a\x6d\x7e\x08\x4b\x35\xc5\xaa\xd8\x40\x66\x8c\x26\x2c\xfa\x4e\xf3\x68\xb1\x62\x8d\xdf\x5a\x69\x41\xa5\xc9\x9f\x84\x24\x94\x5a\x15\x81\xea\xc8\x5a\x63\x47\x72\x30\x76\xe7\x39\xc4\x5d\x6e\x09\x1d\x49\x81\x0e\x78\x7c\xcb\x56\x3c\xba\xbd\xeb\xc7\x99\xdd\xa7\x29\x4b\xb8\x38\x98\x34\xce\xbe\x94\xe7\x38\xd7\xbe\xd3\xeb\xc9\x64\x6c\x1a\xc5\x0b\x16\x6a\x4d\x95\x6b\xa7\xb2\x79\x0d\xce\x66\x6b\x0e\xd9\xd6\x11\x1e\x32\xf9\xfc\x7f\x0e\x37\xbd\xaf\x38\x7b\xe0\x27\xa4\x3d\xd3\x76\xe8\xf9\x15\xb5\xa7\x21\xe0\x6c\x92\x6b\xef\xa3\x32\x2f\xdb\xac\xae\x76\x39\x4c\x9b\xe8\xb3\x65\xb2\xe2\x69\x14\x27\x1c\xd6\xcf\xe2\xbd\x10\x71\x28\x61\xbe\x4c\x59\xfc\x25\xe9\x44\x98\x42\xca\xe6\x2c\x65\xc9\x8c\x1d\x76\xff\x52\xc9\xe9\x64\x7a\x3d\x01\xf8\x90\xab\xca\x19\x8b\x1b\x12\x96\xbe\xfb\xa3\xe6\x04\xdb\x22\xd1\x61\x68\x77\x3b\x1d\xed\x28\x4c\xf5\x82\x5a\xf7\xc7\x7d\x3f\xca\x35\xcb\x85\x71\x50\x78\xad\x5b\xfd\x69\x82\xe5\xc6\x17\xae\x23\x31\x07\x8a\xff\x3b\x5a\x19\x91\xa3\xaf\x68\x6c\xd3\xff\xc0\x7a\x84\xbb\xfa\xbb\xad\xfc\xc4\xdc\xef\x74\x79\x78\xbb\xba\xb4\xa3\xcf\x77\xca\x1d\x40\x33\x25\x03\x48\x3b\x2b\xa5\x35\xaf\x4a\x92\x0d\x98\xd5\xe7\x41\xc9\x61\x85\xab\x41\x7a\x2b\xd5\xfe\x7a\x84\x65\xf0\xdf\xe9\xe9\xbe\xa5\x71\x72\xc3\x1e\x86\x5a\x2a\x86\x88\x5f\x26\x6d\xbf\x07\xd0\x91\x41\xf1\x45\xa9\x0a\xf1\xcd\x64\xa3\xeb\x4e\x6f\x94\x7b\x77\x7c\xe6\x06\xae\x5b\x40\x0e\xdc\xb6\xa4\xd0\xe6\x5b\x42\xb9\x3d\x8f\xf8\xbf\xf5\xca\x99\x92\x9a\x99\x3e\x22\xb5\x56\xb6\x1e\xa9\x67\x56\xd0\x26\xf1\x2b\x00\x00\xff\xff\xa5\xf5\x14\x36\xf8\x09\x00\x00")

func _001_initUpSqlBytes() ([]byte, error) {
	return bindataRead(
		__001_initUpSql,
		"001_init.up.sql",
	)
}

func _001_initUpSql() (*asset, error) {
	bytes, err := _001_initUpSqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "001_init.up.sql", size: 2552, mode: os.FileMode(436), modTime: time.Unix(1628018891, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var __002_rwDownSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x72\x09\xf2\x0f\x50\x08\x71\x74\xf2\x71\x55\x48\x4a\x2c\x49\xce\x88\x2f\x4a\xcd\xcd\x2f\x49\x8d\x2f\x4f\xcc\xc9\x49\x2d\xb1\xe6\x02\x04\x00\x00\xff\xff\x6f\x0a\x2f\x64\x20\x00\x00\x00")

func _002_rwDownSqlBytes() ([]byte, error) {
	return bindataRead(
		__002_rwDownSql,
		"002_rw.down.sql",
	)
}

func _002_rwDownSql() (*asset, error) {
	bytes, err := _002_rwDownSqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "002_rw.down.sql", size: 32, mode: os.FileMode(436), modTime: time.Unix(1631882118, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var __002_rwUpSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x9c\xd0\xc1\x4a\xc3\x40\x10\x06\xe0\x7b\x9f\x62\x8e\x09\xf8\x06\x9e\xd6\x38\x91\xc5\x64\x53\x26\x13\xb0\x88\x2c\x6b\x77\xa4\xa1\xa9\x2d\xe9\x04\x7d\x7c\x49\x52\x73\x90\x9c\xbc\xee\x37\xff\xbf\xf0\x67\x84\x86\x11\xd8\x3c\x14\x08\x36\x07\x57\x31\xe0\x8b\xad\xb9\x86\xf7\xa0\xfb\x83\xef\xe5\x74\x56\xf1\x5f\xa1\xeb\x44\x21\xd9\x00\xc0\x4d\xda\x08\x2a\xdf\x0a\x5b\xb2\xa5\xa1\x1d\x3c\xe3\xee\x6e\xe2\x8b\x48\xbf\xe8\x58\xe8\x9a\xa2\x98\x29\x0c\x7a\xf0\x7a\x3e\xca\xe7\x9a\xce\x9f\xf8\x10\x63\xbf\xc6\xa7\xa1\xd3\x76\xc4\xeb\xa4\xaf\x6f\x7f\x7c\xdf\x4b\x50\x89\x3e\x28\xb0\x2d\xb1\x66\x53\x6e\x97\x13\x78\xc4\xdc\x34\x05\x43\xd6\x10\xa1\x63\xbf\x9c\xcc\xe1\xe1\x12\xff\x17\x9e\xd2\x59\xe5\x6a\x26\x63\x1d\xc3\xc7\xd1\xaf\x2c\xe7\x97\xcd\xf2\x8a\xd0\x3e\xb9\x71\xae\xe4\xf7\x31\x05\xc2\x1c\x09\x5d\x86\xb7\xdd\xe5\x9a\xb4\x31\x9d\xba\xd3\xfb\xcd\xe6\x27\x00\x00\xff\xff\xee\x70\xf3\x84\xa7\x01\x00\x00")

func _002_rwUpSqlBytes() ([]byte, error) {
	return bindataRead(
		__002_rwUpSql,
		"002_rw.up.sql",
	)
}

func _002_rwUpSql() (*asset, error) {
	bytes, err := _002_rwUpSqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "002_rw.up.sql", size: 423, mode: os.FileMode(436), modTime: time.Unix(1631882118, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var __003_providersDownSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x72\xf4\x09\x71\x0d\x52\x08\x71\x74\xf2\x71\x55\x48\x4a\x2c\x49\xce\x48\x2d\x56\x70\x09\xf2\x0f\x50\x70\xf6\xf7\x09\xf5\xf5\x53\x28\x28\xca\x2f\xcb\x4c\x49\x2d\x2a\xb6\xe6\x02\x04\x00\x00\xff\xff\x21\xd6\x22\x59\x2b\x00\x00\x00")

func _003_providersDownSqlBytes() ([]byte, error) {
	return bindataRead(
		__003_providersDownSql,
		"003_providers.down.sql",
	)
}

func _003_providersDownSql() (*asset, error) {
	bytes, err := _003_providersDownSqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "003_providers.down.sql", size: 43, mode: os.FileMode(436), modTime: time.Unix(1633021059, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var __003_providersUpSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x72\xf4\x09\x71\x0d\x52\x08\x71\x74\xf2\x71\x55\x48\x4a\x2c\x49\xce\x48\x2d\x56\x70\x74\x71\x51\x70\xf6\xf7\x09\xf5\xf5\x53\xf0\x74\x53\xf0\xf3\x0f\x51\x70\x8d\xf0\x0c\x0e\x09\x56\x28\x28\xca\x2f\xcb\x4c\x49\x2d\x2a\x56\x08\x71\x8d\x08\x89\x8e\xb5\xe6\x02\x04\x00\x00\xff\xff\x81\xdd\xbf\xe1\x3f\x00\x00\x00")

func _003_providersUpSqlBytes() ([]byte, error) {
	return bindataRead(
		__003_providersUpSql,
		"003_providers.up.sql",
	)
}

func _003_providersUpSql() (*asset, error) {
	bytes, err := _003_providersUpSqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "003_providers.up.sql", size: 63, mode: os.FileMode(436), modTime: time.Unix(1633021059, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var __004_status_enumsDownSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x94\xd2\xbb\x6e\x83\x30\x14\x80\xe1\xdd\x4f\x71\x36\xd6\xde\x17\x94\x81\x04\xab\x8d\xc4\x4d\xd4\xa8\xea\x84\x5c\x6a\xa5\x28\xc4\xa4\xb6\x51\xfb\xf8\x15\x24\x34\x44\xc6\xc2\x5e\xe1\x9c\xdf\x60\x7d\x6b\xfc\xbc\x4d\x7c\x84\x82\x88\xe0\x1c\x48\xb0\x8e\x30\x48\xd5\x0a\xba\x63\xa5\x60\xdf\x1d\x93\x4a\x42\x10\x86\xb0\x49\xa3\x22\x4e\x40\x2a\xaa\x3a\x59\xd6\x5c\x81\x3c\xd0\xa6\xa9\xb9\xf2\x11\x2a\xb2\x30\x20\x33\x8b\xaf\x98\x4c\x37\x56\x70\x03\x6f\x2f\x38\xc7\xe7\x87\xb0\x02\xaf\xe3\x7b\xde\xfe\x70\xcf\xb7\x8f\xdc\x6a\x91\x0f\xaa\xaa\xaf\x9a\xef\x5c\x2a\x77\x5a\xe5\x28\xd8\x91\x0a\xc7\xcc\xbd\x96\xa1\x5d\xa5\xea\x96\x3b\x76\x1e\xb4\xce\x27\xa3\x4d\x79\xa0\x7b\xc7\xd0\xa3\x16\x92\x5d\x55\x31\x29\x5d\x22\x4f\x5a\x84\x09\xd1\x0a\x6f\x49\x4a\x98\xa7\xd9\x35\x95\xa5\x8d\x1c\x27\x41\x8c\x67\x78\x91\xd4\xb2\x70\x7a\x79\x15\x18\x7e\x28\x49\x09\x24\x45\x14\xf9\x08\x0d\x9f\x45\xde\x33\x6d\xbb\x9c\x3d\x62\xe0\xc4\xe6\xdc\x4b\x25\x40\xb1\xdf\x89\xf9\x71\x76\x72\x85\xfd\xd0\x44\xf6\x32\x79\x73\xe3\x42\xd2\x46\xab\xb9\x33\x31\x69\xe5\xd5\x5c\xea\x55\xc2\x59\xa5\x1d\x59\x73\x6b\x84\xb9\x2c\xd6\xdc\x38\xb9\xb4\xe3\x3a\x56\x16\x95\x8e\x83\xb3\x38\xfb\x83\x4d\x38\xff\xe5\xb8\x98\x1c\x96\x2e\x12\x37\x69\x1c\x6f\x89\x8f\xfe\x02\x00\x00\xff\xff\xa5\x47\x2e\x9b\x9b\x05\x00\x00")

func _004_status_enumsDownSqlBytes() ([]byte, error) {
	return bindataRead(
		__004_status_enumsDownSql,
		"004_status_enums.down.sql",
	)
}

func _004_status_enumsDownSql() (*asset, error) {
	bytes, err := _004_status_enumsDownSqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "004_status_enums.down.sql", size: 1435, mode: os.FileMode(436), modTime: time.Unix(1633541920, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var __004_status_enumsUpSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xa4\x93\x5d\x6b\xab\x30\x18\xc7\xef\xfd\x14\xcf\x9d\xe7\xc0\xb9\x38\xe7\xec\xe5\x46\x0a\xb3\x35\x6c\x05\x8d\xa2\x91\x31\x18\x48\x66\x43\x57\xda\xc6\x2e\x51\xc6\xbe\xfd\xf0\xa5\xd6\xa6\xb5\x4d\xd8\x55\x4d\xff\xc9\x2f\x79\x92\xe7\x37\x45\x8f\x73\xec\x58\xd6\x2c\x46\x2e\x41\x40\x5e\x22\x04\xb2\x2c\x04\x5d\xb2\x4c\xb0\x8f\x8a\xc9\x32\x93\x25\x2d\x2b\x09\x6e\x02\x08\xa7\x01\xfc\xb2\x2b\xbe\xe6\xc5\x27\xb7\xff\x80\xfd\x46\xcb\xfc\x7d\xc5\x97\xf5\xf7\x4e\xb0\x1d\x15\xdd\x80\x56\x79\xb9\x2a\x78\x37\x5a\x30\xba\xc9\xb6\x74\xdd\x0d\x65\x95\xe7\x4c\xca\xfa\x93\x09\x51\x08\xfb\xb7\x63\x59\xae\x4f\x50\x0c\xc4\x9d\xfa\x27\x47\x90\xe0\x7a\x1e\xcc\x42\x3f\x0d\x30\xb4\xc7\xc9\x18\xaf\xb6\x23\x47\x75\x2c\x2b\x8d\xbc\xba\x9e\x13\x4e\x82\xc8\x11\x60\x02\x7d\x35\xf0\xfc\x84\x62\xd4\xa5\x30\x81\xbf\x8e\x09\xa5\xbf\x08\x15\xf3\xcf\x08\x73\xb8\x43\x95\xf3\xdf\x88\x33\xb8\x7e\x15\x74\x63\x04\x1a\xbe\x9c\x4a\xba\x35\x22\xed\x1f\x5d\xa5\xdc\x19\x51\xda\x7e\x51\x19\xf7\xd7\xfa\xc7\x8b\xc3\xe8\xb8\x81\xae\xad\x88\x11\x76\x03\x74\xae\xe9\x48\xa8\x89\x68\xc3\x23\x42\x53\x12\x0e\x09\xe0\xd4\xf7\x15\xef\x9a\x16\xba\x68\xdb\xcf\x0c\x9b\x85\x41\x80\x30\x81\x10\x9f\xd9\x6f\x9e\x00\xb2\x1f\xea\xfa\x5e\x79\xf3\x83\xe9\x96\x41\xd2\x96\x14\xd1\xaf\x4d\x41\x17\x49\x33\xd5\x56\xca\x6e\x30\x6c\x54\xd1\xe1\x2e\x07\x31\xf7\x8b\x0c\x7c\xec\x03\x47\x03\x32\xee\xd1\x20\xd2\x01\x5d\x10\x69\x98\xe9\xa0\x2e\xa9\xd4\x84\xd0\x85\x3a\xb0\x31\x9b\xfa\x40\x07\x72\x5e\xa6\xee\xef\x91\x67\xbe\x6a\xd2\x7e\xa2\xb9\x40\x7d\x27\x69\x78\x13\x06\xc1\x9c\x38\xd6\x77\x00\x00\x00\xff\xff\x23\x54\x6f\xf6\xc0\x06\x00\x00")

func _004_status_enumsUpSqlBytes() ([]byte, error) {
	return bindataRead(
		__004_status_enumsUpSql,
		"004_status_enums.up.sql",
	)
}

func _004_status_enumsUpSql() (*asset, error) {
	bytes, err := _004_status_enumsUpSqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "004_status_enums.up.sql", size: 1728, mode: os.FileMode(436), modTime: time.Unix(1633541920, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var __005_payload_sizeDownSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x72\xf4\x09\x71\x0d\x52\x08\x71\x74\xf2\x71\x55\x48\x4a\x2c\x49\xce\x48\x2d\x56\x70\x09\xf2\x0f\x50\x70\xf6\xf7\x09\xf5\xf5\x53\x28\x48\xac\xcc\xc9\x4f\x4c\x89\x2f\xce\xac\x4a\xb5\xe6\x02\x04\x00\x00\xff\xff\xb6\x96\xa3\xfc\x2e\x00\x00\x00")

func _005_payload_sizeDownSqlBytes() ([]byte, error) {
	return bindataRead(
		__005_payload_sizeDownSql,
		"005_payload_size.down.sql",
	)
}

func _005_payload_sizeDownSql() (*asset, error) {
	bytes, err := _005_payload_sizeDownSqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "005_payload_size.down.sql", size: 46, mode: os.FileMode(436), modTime: time.Unix(1633611889, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var __005_payload_sizeUpSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x72\xf4\x09\x71\x0d\x52\x08\x71\x74\xf2\x71\x55\x48\x4a\x2c\x49\xce\x48\x2d\x56\x70\x74\x71\x51\x70\xf6\xf7\x09\xf5\xf5\x53\x28\x48\xac\xcc\xc9\x4f\x4c\x89\x2f\xce\xac\x4a\x55\x48\xca\x4c\xcf\xcc\x2b\xb1\xe6\x02\x04\x00\x00\xff\xff\x99\x48\x68\x88\x34\x00\x00\x00")

func _005_payload_sizeUpSqlBytes() ([]byte, error) {
	return bindataRead(
		__005_payload_sizeUpSql,
		"005_payload_size.up.sql",
	)
}

func _005_payload_sizeUpSql() (*asset, error) {
	bytes, err := _005_payload_sizeUpSqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "005_payload_size.up.sql", size: 52, mode: os.FileMode(436), modTime: time.Unix(1633611889, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var __006_add_dealstartoffsetDownSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x72\xf4\x09\x71\x0d\x52\x08\x71\x74\xf2\x71\x55\x48\x4a\x2c\x49\xce\x48\x2d\x56\x70\x09\xf2\x0f\x50\x70\xf6\xf7\x09\xf5\xf5\x53\xf0\x74\x53\x70\x8d\xf0\x0c\x0e\x09\x56\x28\x28\xca\x2f\xc8\x2f\x4e\xcc\x89\x2f\x2e\x49\x2c\x2a\x89\xcf\x4f\x4b\x2b\x4e\x2d\x89\x2f\x4e\x4d\xce\xcf\x4b\x29\xb6\xe6\x02\x04\x00\x00\xff\xff\x14\xa2\x6b\x0f\x49\x00\x00\x00")

func _006_add_dealstartoffsetDownSqlBytes() ([]byte, error) {
	return bindataRead(
		__006_add_dealstartoffsetDownSql,
		"006_add_dealstartoffset.down.sql",
	)
}

func _006_add_dealstartoffsetDownSql() (*asset, error) {
	bytes, err := _006_add_dealstartoffsetDownSqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "006_add_dealstartoffset.down.sql", size: 73, mode: os.FileMode(436), modTime: time.Unix(1634734442, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var __006_add_dealstartoffsetUpSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x7c\xcc\xbd\x0a\xc2\x30\x10\x00\xe0\x3d\x4f\x71\x8f\x50\x5d\x14\x82\x43\xda\xc4\x72\x90\x1f\xb1\x17\x70\x0b\xb1\xa6\x38\x88\x29\xbd\xbc\x3f\x82\x8b\x9b\xfb\xc7\xd7\x9b\x11\xbd\x14\xca\x92\xb9\x02\xa9\xde\x1a\xb8\xe7\x36\x3f\x0b\x83\xd2\x1a\x86\x60\xa3\xf3\x80\x67\xf0\x81\xc0\xdc\x70\xa2\x09\xd6\xad\xae\x95\xf3\x2b\x71\xcb\x5b\x4b\x75\x59\xb8\xb4\xc4\x65\xae\xef\x07\x43\x8f\x23\x7a\xfa\x7a\x1f\xad\x95\x22\x5e\xb4\xa2\x5f\xcb\xa5\xfd\x1f\x4e\xbb\xc3\xfe\xd8\x75\x52\x0c\xc1\x39\x24\x29\x3e\x01\x00\x00\xff\xff\x96\xdd\x3c\x51\xa4\x00\x00\x00")

func _006_add_dealstartoffsetUpSqlBytes() ([]byte, error) {
	return bindataRead(
		__006_add_dealstartoffsetUpSql,
		"006_add_dealstartoffset.up.sql",
	)
}

func _006_add_dealstartoffsetUpSql() (*asset, error) {
	bytes, err := _006_add_dealstartoffsetUpSqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "006_add_dealstartoffset.up.sql", size: 164, mode: os.FileMode(436), modTime: time.Unix(1634736555, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

// Asset loads and returns the asset for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func Asset(name string) ([]byte, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("Asset %s can't read by error: %v", name, err)
		}
		return a.bytes, nil
	}
	return nil, fmt.Errorf("Asset %s not found", name)
}

// MustAsset is like Asset but panics when Asset would return an error.
// It simplifies safe initialization of global variables.
func MustAsset(name string) []byte {
	a, err := Asset(name)
	if err != nil {
		panic("asset: Asset(" + name + "): " + err.Error())
	}

	return a
}

// AssetInfo loads and returns the asset info for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func AssetInfo(name string) (os.FileInfo, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("AssetInfo %s can't read by error: %v", name, err)
		}
		return a.info, nil
	}
	return nil, fmt.Errorf("AssetInfo %s not found", name)
}

// AssetNames returns the names of the assets.
func AssetNames() []string {
	names := make([]string, 0, len(_bindata))
	for name := range _bindata {
		names = append(names, name)
	}
	return names
}

// _bindata is a table, holding each asset generator, mapped to its name.
var _bindata = map[string]func() (*asset, error){
	"001_init.down.sql":                _001_initDownSql,
	"001_init.up.sql":                  _001_initUpSql,
	"002_rw.down.sql":                  _002_rwDownSql,
	"002_rw.up.sql":                    _002_rwUpSql,
	"003_providers.down.sql":           _003_providersDownSql,
	"003_providers.up.sql":             _003_providersUpSql,
	"004_status_enums.down.sql":        _004_status_enumsDownSql,
	"004_status_enums.up.sql":          _004_status_enumsUpSql,
	"005_payload_size.down.sql":        _005_payload_sizeDownSql,
	"005_payload_size.up.sql":          _005_payload_sizeUpSql,
	"006_add_dealstartoffset.down.sql": _006_add_dealstartoffsetDownSql,
	"006_add_dealstartoffset.up.sql":   _006_add_dealstartoffsetUpSql,
}

// AssetDir returns the file names below a certain
// directory embedded in the file by go-bindata.
// For example if you run go-bindata on data/... and data contains the
// following hierarchy:
//     data/
//       foo.txt
//       img/
//         a.png
//         b.png
// then AssetDir("data") would return []string{"foo.txt", "img"}
// AssetDir("data/img") would return []string{"a.png", "b.png"}
// AssetDir("foo.txt") and AssetDir("notexist") would return an error
// AssetDir("") will return []string{"data"}.
func AssetDir(name string) ([]string, error) {
	node := _bintree
	if len(name) != 0 {
		cannonicalName := strings.Replace(name, "\\", "/", -1)
		pathList := strings.Split(cannonicalName, "/")
		for _, p := range pathList {
			node = node.Children[p]
			if node == nil {
				return nil, fmt.Errorf("Asset %s not found", name)
			}
		}
	}
	if node.Func != nil {
		return nil, fmt.Errorf("Asset %s not found", name)
	}
	rv := make([]string, 0, len(node.Children))
	for childName := range node.Children {
		rv = append(rv, childName)
	}
	return rv, nil
}

type bintree struct {
	Func     func() (*asset, error)
	Children map[string]*bintree
}

var _bintree = &bintree{nil, map[string]*bintree{
	"001_init.down.sql":                &bintree{_001_initDownSql, map[string]*bintree{}},
	"001_init.up.sql":                  &bintree{_001_initUpSql, map[string]*bintree{}},
	"002_rw.down.sql":                  &bintree{_002_rwDownSql, map[string]*bintree{}},
	"002_rw.up.sql":                    &bintree{_002_rwUpSql, map[string]*bintree{}},
	"003_providers.down.sql":           &bintree{_003_providersDownSql, map[string]*bintree{}},
	"003_providers.up.sql":             &bintree{_003_providersUpSql, map[string]*bintree{}},
	"004_status_enums.down.sql":        &bintree{_004_status_enumsDownSql, map[string]*bintree{}},
	"004_status_enums.up.sql":          &bintree{_004_status_enumsUpSql, map[string]*bintree{}},
	"005_payload_size.down.sql":        &bintree{_005_payload_sizeDownSql, map[string]*bintree{}},
	"005_payload_size.up.sql":          &bintree{_005_payload_sizeUpSql, map[string]*bintree{}},
	"006_add_dealstartoffset.down.sql": &bintree{_006_add_dealstartoffsetDownSql, map[string]*bintree{}},
	"006_add_dealstartoffset.up.sql":   &bintree{_006_add_dealstartoffsetUpSql, map[string]*bintree{}},
}}

// RestoreAsset restores an asset under the given directory
func RestoreAsset(dir, name string) error {
	data, err := Asset(name)
	if err != nil {
		return err
	}
	info, err := AssetInfo(name)
	if err != nil {
		return err
	}
	err = os.MkdirAll(_filePath(dir, filepath.Dir(name)), os.FileMode(0755))
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(_filePath(dir, name), data, info.Mode())
	if err != nil {
		return err
	}
	err = os.Chtimes(_filePath(dir, name), info.ModTime(), info.ModTime())
	if err != nil {
		return err
	}
	return nil
}

// RestoreAssets restores an asset under the given directory recursively
func RestoreAssets(dir, name string) error {
	children, err := AssetDir(name)
	// File
	if err != nil {
		return RestoreAsset(dir, name)
	}
	// Dir
	for _, child := range children {
		err = RestoreAssets(dir, filepath.Join(name, child))
		if err != nil {
			return err
		}
	}
	return nil
}

func _filePath(dir, name string) string {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	return filepath.Join(append([]string{dir}, strings.Split(cannonicalName, "/")...)...)
}
