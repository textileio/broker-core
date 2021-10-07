// Code generated by go-bindata. (@generated) DO NOT EDIT.

 //Package migrations generated by go-bindata.// sources:
// migrations/001_init.down.sql
// migrations/001_init.up.sql
// migrations/002_market_status.down.sql
// migrations/002_market_status.up.sql
// migrations/003_remote_wallet.down.sql
// migrations/003_remote_wallet.up.sql
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

var __001_initDownSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x72\x09\xf2\x0f\x50\x08\x71\x74\xf2\x71\x55\x48\x2c\x4d\x2e\xc9\xcc\xcf\x8b\x4f\x49\x4d\xcc\x29\xb6\xe6\x82\xc8\x44\x06\xb8\x2a\x14\x97\x24\x96\x94\xc2\x45\x50\xd5\x26\x96\x24\x5a\x73\x01\x02\x00\x00\xff\xff\x00\x02\xb7\x1e\x45\x00\x00\x00")

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

	info := bindataFileInfo{name: "001_init.down.sql", size: 69, mode: os.FileMode(436), modTime: time.Unix(1628018891, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var __001_initUpSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xa4\x94\xcf\x6e\xe2\x30\x10\xc6\xef\x3c\xc5\xdc\x00\x69\xfb\x04\x3d\x65\xa9\x59\x45\x0b\x01\x85\x20\x95\x93\x35\xd8\x03\x1d\x11\xe2\x68\xec\x20\xda\xa7\x5f\x25\x94\xaa\xcd\x26\x59\x69\x7b\xb2\x94\xdf\xe7\xf1\xfc\xf9\x26\xb3\x54\x45\x99\x82\x2c\xfa\xb9\x50\x10\xcf\x21\x59\x65\xa0\x9e\xe3\x4d\xb6\x01\xac\x4c\x60\x57\x68\x8b\x01\x61\x32\x02\x00\x60\x0b\x81\xae\x01\xd6\x69\xbc\x8c\xd2\x1d\xfc\x56\xbb\x1f\x0d\xd8\x63\x30\x2f\xfa\x8e\xeb\x20\xc9\x76\xb1\xb8\xb1\x12\x5f\x73\x87\x56\x9b\x1e\xcc\x64\x68\x18\x7a\x7e\x23\xd8\xf3\x91\x8b\x36\xb7\x95\x60\x9d\x64\x37\x35\x42\x18\xc8\x6a\x0c\x90\xc5\x4b\xb5\xc9\xa2\xe5\xfa\x43\x02\x4f\x6a\x1e\x6d\x17\x19\xcc\xb6\x69\xaa\x92\x4c\x7f\x48\x9a\xbb\xd3\xc7\xd1\xe8\xde\x9c\xdd\x5a\x81\x0f\x18\x2a\x0f\xd1\x06\x54\xb2\x5d\x36\xfd\x18\x5b\xc2\xfc\xe1\x8c\x27\x2e\x8e\xe3\xfa\xc5\xb1\x71\xc5\x81\xe5\xdc\xa4\x74\xfb\x22\x54\x3a\x09\x0f\x07\x2e\x30\xe7\x37\xb2\xe3\xd1\xe7\xc0\x43\x5d\x27\xcc\xfd\xbf\xda\xfe\x79\x44\x3d\xdd\xf7\xc1\x09\x1e\x49\x97\xe2\x2e\x6c\x49\xfa\x86\x24\x6c\x48\x97\x24\xfa\xc8\xfb\xe6\xa4\xd2\x99\x97\xee\xbe\xfa\x80\x12\x86\x04\x17\x12\x3e\x30\x59\xd8\x3b\x97\x13\x16\x2d\x7c\x40\x1f\xb4\x50\x10\xa6\x0b\xe6\x3d\xa2\x7b\x6d\xdd\xf9\xee\xd9\xf6\x16\xdc\x0c\xea\xfd\xf8\xca\xe8\x4a\xa6\x0a\x5c\x1c\x7b\xde\x24\x11\x27\xda\x60\xe5\xa9\x2b\xf4\x2d\x65\x0f\xfe\x8c\x79\xfe\x77\xd5\xa5\xb8\xd2\x79\xcc\xfb\xcc\x5c\x8f\xb4\x4e\xba\xdb\xc9\x35\xa4\x6b\xc9\x43\x86\x6e\x44\x67\x94\x13\x05\xfd\x5e\x60\xa7\x4e\x08\xed\x6b\xb7\xed\xbf\xb5\x19\xb7\xcb\x55\x69\xff\xff\xf2\x6c\x95\x6c\xb2\x34\x8a\x93\x0c\x0e\x27\xdd\x36\xf0\x7c\x95\xaa\xf8\x57\x52\x3b\x7c\xd2\x62\x53\x48\xd5\x5c\xa5\x2a\x99\xa9\xaf\xff\xa6\x09\xdb\x69\x7b\x65\xe3\xe4\x49\x3d\x0f\x6d\x96\xee\xda\x8b\x55\xd2\x5e\xbf\x0e\xd5\xf4\x71\xf4\x27\x00\x00\xff\xff\xc9\xf0\x8a\x40\x35\x05\x00\x00")

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

	info := bindataFileInfo{name: "001_init.up.sql", size: 1333, mode: os.FileMode(436), modTime: time.Unix(1628018891, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var __002_market_statusDownSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x72\x09\xf2\x0f\x50\x08\x71\x74\xf2\x71\x55\xc8\x4d\x2c\xca\x4e\x2d\x89\x4f\x49\x4d\xcc\x89\x2f\x2e\x49\x2c\x29\x2d\xb6\xe6\xe2\xe2\x02\x04\x00\x00\xff\xff\x14\xa7\xbe\x6d\x21\x00\x00\x00")

func _002_market_statusDownSqlBytes() ([]byte, error) {
	return bindataRead(
		__002_market_statusDownSql,
		"002_market_status.down.sql",
	)
}

func _002_market_statusDownSql() (*asset, error) {
	bytes, err := _002_market_statusDownSqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "002_market_status.down.sql", size: 33, mode: os.FileMode(436), modTime: time.Unix(1628112604, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var __002_market_statusUpSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x6c\x92\x41\x6f\xda\x4e\x10\xc5\xcf\x7f\x7f\x8a\xb9\x19\x24\x1f\xfe\xa6\x25\x49\x95\x93\x8b\x8c\x84\x4a\x09\x02\x43\x9b\x53\x34\xb1\x07\x98\x62\x76\xd1\xee\x02\x69\x3f\x7d\x35\x86\x78\xd7\x6d\xaf\xef\xbd\x7d\x33\xf3\xb3\x47\x8b\x3c\x2b\x72\x28\xb2\xcf\xd3\x1c\x26\x63\x98\x3d\x15\x90\x7f\x9f\x2c\x8b\x25\x1c\xd0\xec\xc9\xbd\x54\x84\xf5\x8b\x75\xe8\x4e\x16\x7a\x11\x00\x00\x57\xf0\xca\x5b\x56\xae\x49\xcf\x56\xd3\x29\xcc\x17\x93\xaf\xd9\xe2\x19\xbe\xe4\xcf\x49\x93\x51\x78\x20\x70\xf4\xe6\x33\x51\xff\x31\x8a\x26\xb3\x65\xbe\x28\x60\x32\x2b\x9e\xfe\xd9\xcf\x55\xd2\x3c\xed\x83\xb4\xac\xb3\xe9\x2a\x5f\x36\x7d\xbd\xff\x13\x88\x57\x6a\xaf\xf4\x45\xc5\xfd\xeb\x8c\x5e\x9a\x40\x3c\x37\xfa\xa8\x2d\xd6\xa0\xb4\x83\x8d\x3e\xa9\xaa\xb5\x07\xa1\x6d\xe8\x07\x95\x8e\xbc\xfb\x21\x74\xb1\x2c\xe9\x18\xba\x1f\x13\x88\x97\x0e\xb7\x81\x34\x14\x89\xb0\x66\xb5\x6d\xb5\xbb\x04\xe2\x31\x2b\xac\xf9\x57\x28\xdf\x27\x10\x67\xa5\xe3\x33\xb5\xd2\x43\x02\x71\xfe\x76\x64\x13\x34\x7e\x92\xc6\x1a\xed\x2e\xd0\x52\x39\x74\xd1\x2c\x1b\x36\xa6\x72\xeb\x18\xb9\x33\x3d\x95\x0b\xc7\x27\x55\x59\x30\x64\xc9\x9c\xc3\x1e\xb9\x6f\xb4\xa3\x72\xcf\x6a\x0b\x1b\x6d\x40\x40\xdf\x0e\x45\x55\xfa\xcd\x52\xb9\x75\x8d\x35\x57\xd8\x1d\x39\x6c\xae\x90\x3c\x5c\x90\x9d\x37\xee\xae\x70\x8c\xc4\xa1\x42\x87\xe0\x0c\x2a\xbb\x21\xe3\x33\x82\xa0\xb8\xa9\xa6\x53\x2b\x24\xbe\x21\xbb\x76\x2f\x74\xe8\x5d\x61\xb2\x26\xc3\x9b\x9f\xef\xe5\xfe\x7b\x5e\xd1\xc8\xa1\xe2\x1d\x8d\x3e\x73\x45\x06\x36\x42\xc0\xa7\xd2\x4e\xaa\xac\x99\x94\xfb\x33\x73\xfb\x35\xfc\xfb\x70\xc3\x41\x83\xce\xbf\xeb\x78\x02\x6b\x7e\x7a\xad\xd9\xee\xbc\x38\xf4\x62\x27\x2c\xa0\x72\x63\xb4\xe7\x32\xb8\x0f\x27\xbf\x63\x03\x14\xc0\xf2\x11\x85\xaa\x0f\x3f\xf8\x45\xda\xe8\x5f\x21\x21\x96\x5d\x6e\x40\x11\xe6\x86\x46\xfa\x70\x60\x07\x07\xb2\x16\xb7\x04\x5a\x41\xb9\x43\x56\x71\xff\x31\xfa\x2f\xfa\x1d\x00\x00\xff\xff\x74\xb3\xf8\x95\xf2\x03\x00\x00")

func _002_market_statusUpSqlBytes() ([]byte, error) {
	return bindataRead(
		__002_market_statusUpSql,
		"002_market_status.up.sql",
	)
}

func _002_market_statusUpSql() (*asset, error) {
	bytes, err := _002_market_statusUpSqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "002_market_status.up.sql", size: 1010, mode: os.FileMode(436), modTime: time.Unix(1628112604, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var __003_remote_walletDownSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x72\x09\xf2\x0f\x50\x08\x71\x74\xf2\x71\x55\x28\x4a\xcd\xcd\x2f\x49\x8d\x2f\x4f\xcc\xc9\x49\x2d\xb1\xe6\x02\x04\x00\x00\xff\xff\x6a\x10\xc8\xed\x1a\x00\x00\x00")

func _003_remote_walletDownSqlBytes() ([]byte, error) {
	return bindataRead(
		__003_remote_walletDownSql,
		"003_remote_wallet.down.sql",
	)
}

func _003_remote_walletDownSql() (*asset, error) {
	bytes, err := _003_remote_walletDownSqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "003_remote_wallet.down.sql", size: 26, mode: os.FileMode(436), modTime: time.Unix(1631882118, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var __003_remote_walletUpSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x9c\x90\xcf\x6a\x83\x40\x10\x87\xef\x3e\xc5\x1c\x15\xf2\x06\x3d\x6d\xed\x58\x96\xea\x1a\x76\x47\x68\x28\x65\x59\xb2\x53\x2a\x31\x31\xd8\x91\xf6\xf1\x8b\xa6\x08\x11\x4f\x3d\x7f\xbf\x3f\xf0\xe5\x16\x15\x21\x90\x7a\x2c\x11\x74\x01\xa6\x26\xc0\x57\xed\xc8\xc1\xc0\xe7\x5e\xd8\x7f\x87\xae\x63\x81\x34\x01\x00\x08\xe3\x51\xda\xfe\xe2\x63\x90\xe0\xdb\x08\xc2\x3f\x02\x7b\xab\x2b\x65\x0f\xf0\x82\x87\xdd\x9c\xba\x32\x0f\x0b\x9d\x16\x4d\x53\x96\xbb\xbf\x01\xf9\xf4\xd2\x9f\xf8\xb2\x45\x6f\x5f\x3e\xc4\x38\x6c\xe1\xf3\xd8\x49\x3b\xc1\xaf\x99\xbe\xbd\xaf\xf8\x71\xe0\x20\x1c\x7d\x10\x20\x5d\xa1\x23\x55\xed\x97\x08\x3c\x61\xa1\x9a\x92\x20\x6f\xac\x45\x43\x7e\x89\xdc\xca\xe3\x35\xfe\xaf\x3c\xb7\xf3\xda\x38\xb2\x4a\x1b\x82\x8f\x93\xbf\x53\xe7\xd7\xd2\x8a\xda\xa2\x7e\x36\x93\xaf\x74\xc5\x32\xb0\x58\xa0\x45\x93\xa3\xbb\x93\x9d\xb6\x31\x9b\x8f\xb2\x87\x24\xf9\x0d\x00\x00\xff\xff\x37\x93\x40\x7d\xb5\x01\x00\x00")

func _003_remote_walletUpSqlBytes() ([]byte, error) {
	return bindataRead(
		__003_remote_walletUpSql,
		"003_remote_wallet.up.sql",
	)
}

func _003_remote_walletUpSql() (*asset, error) {
	bytes, err := _003_remote_walletUpSqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "003_remote_wallet.up.sql", size: 437, mode: os.FileMode(436), modTime: time.Unix(1631882118, 0)}
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
	"001_init.down.sql":          _001_initDownSql,
	"001_init.up.sql":            _001_initUpSql,
	"002_market_status.down.sql": _002_market_statusDownSql,
	"002_market_status.up.sql":   _002_market_statusUpSql,
	"003_remote_wallet.down.sql": _003_remote_walletDownSql,
	"003_remote_wallet.up.sql":   _003_remote_walletUpSql,
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
	"001_init.down.sql":          &bintree{_001_initDownSql, map[string]*bintree{}},
	"001_init.up.sql":            &bintree{_001_initUpSql, map[string]*bintree{}},
	"002_market_status.down.sql": &bintree{_002_market_statusDownSql, map[string]*bintree{}},
	"002_market_status.up.sql":   &bintree{_002_market_statusUpSql, map[string]*bintree{}},
	"003_remote_wallet.down.sql": &bintree{_003_remote_walletDownSql, map[string]*bintree{}},
	"003_remote_wallet.up.sql":   &bintree{_003_remote_walletUpSql, map[string]*bintree{}},
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
