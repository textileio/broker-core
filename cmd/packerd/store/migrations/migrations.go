// Code generated by go-bindata. (@generated) DO NOT EDIT.

 //Package migrations generated by go-bindata.// sources:
// migrations/001_init.down.sql
// migrations/001_init.up.sql
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

var __001_initDownSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x72\x09\xf2\x0f\x50\x08\x71\x74\xf2\x71\x55\x48\x4a\x2c\x49\xce\x88\x4f\x49\x4c\x2f\xe6\x42\x17\x4d\x2d\xb6\x86\x8a\x45\x06\xc0\x85\xe2\x8b\x4b\x12\x4b\x4a\x8b\xad\xb9\x00\x01\x00\x00\xff\xff\x51\x7a\x82\xf6\x44\x00\x00\x00")

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

	info := bindataFileInfo{name: "001_init.down.sql", size: 68, mode: os.FileMode(420), modTime: time.Unix(1628108979, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var __001_initUpSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xc4\x52\x4d\xaf\x9b\x30\x10\xbc\xf3\x2b\xf6\x66\x90\x38\xf4\xfe\x4e\x3c\xde\x12\x59\x05\x13\x19\x23\x25\x27\xcb\xc5\x4e\x8a\x1a\x41\x0a\x46\x4a\xfb\xeb\xab\x60\x20\x1f\xa5\xaa\x94\xcb\xbb\x70\x60\x76\x66\x3d\x33\x1b\x73\x8c\x04\x82\xd8\x6f\x11\xbe\x29\x5b\x7d\x97\xbd\x55\x76\xe8\x21\x2a\x00\x59\x99\x81\x4f\xda\xb3\x69\x48\x48\x3a\xa3\xf4\x2f\x12\x12\x73\x31\xd5\x60\xeb\xe6\x48\x42\xa2\xdb\xc6\x90\xe0\xcd\xf3\x66\x99\xe8\x3d\x45\xa0\x09\xb0\x5c\x00\xee\x68\x21\x0a\xa7\x6a\x7a\xf0\x3d\x00\x98\x76\xd4\x1a\x04\xee\x04\x6c\x39\xcd\x22\xbe\x87\xaf\xb8\x0f\x47\x78\x5a\xfe\xf0\x92\xab\x16\x2b\xd3\x14\xb4\x39\xa8\xe1\x64\x61\x7a\xd1\x48\xb0\xad\x55\x27\xd9\xd7\xbf\x0d\xbc\xd3\x0d\x65\xe2\x36\xfe\x81\x49\x54\xa6\x02\xbe\xb8\xc9\xb6\xab\x8f\x75\xe3\xf6\xce\x33\x0e\x19\x9d\x49\x65\x41\xd0\x0c\x0b\x11\x65\xdb\xbf\x45\xe2\x92\x73\x64\x42\x2e\x23\x8e\x5a\x75\x46\x59\xa3\x5f\x23\x0f\x67\xfd\x0a\xd9\x0b\xde\xe6\xbc\x29\xfb\xc0\xdd\x7a\xde\x53\x7a\x72\x36\x27\x6b\x7d\x81\x9c\xcd\xb0\xef\xe0\x70\x31\xff\x9f\x16\x7b\xdb\x76\xea\x68\x64\x67\x7e\x0e\xa6\xb7\x73\x9d\xed\xd9\x74\xca\xd6\x6d\xb3\x54\xfa\x18\xed\x13\xed\x1f\x53\x5a\x59\x25\xab\x75\xec\xf1\x60\x9e\xd4\x57\x6a\xff\xbc\x5e\x1c\x79\xfc\xdc\xdd\xb5\x7f\x1f\x51\xb8\x12\x48\xe0\x78\x71\xce\x0a\xc1\xa3\xab\x95\xc3\x0f\xb9\xb8\x4e\x72\x8e\x74\xc3\x46\xa5\xf9\x67\x00\x1c\x13\xe4\xc8\x62\x5c\xea\xbe\x81\xd7\xf3\xf0\xbc\x3f\x01\x00\x00\xff\xff\xb9\xc0\x66\x5e\xd8\x03\x00\x00")

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

	info := bindataFileInfo{name: "001_init.up.sql", size: 984, mode: os.FileMode(420), modTime: time.Unix(1628185127, 0)}
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
	"001_init.down.sql": _001_initDownSql,
	"001_init.up.sql":   _001_initUpSql,
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
	"001_init.down.sql": &bintree{_001_initDownSql, map[string]*bintree{}},
	"001_init.up.sql":   &bintree{_001_initUpSql, map[string]*bintree{}},
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
