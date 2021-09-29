// Code generated by go-bindata. (@generated) DO NOT EDIT.

 //Package migrations generated by go-bindata.// sources:
// migrations/001_init.down.sql
// migrations/001_init.up.sql
// migrations/002_bids_support_calculating_rates.down.sql
// migrations/002_bids_support_calculating_rates.up.sql
// migrations/003_auctions_add_client_address.down.sql
// migrations/003_auctions_add_client_address.up.sql
// migrations/004_bids_add_won_reason.down.sql
// migrations/004_bids_add_won_reason.up.sql
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

var __001_initDownSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x72\x09\xf2\x0f\x50\x08\x71\x74\xf2\x71\x55\x48\xca\x4c\x29\xb6\xe6\x42\x12\x48\x2c\x4d\x2e\xc9\xcc\xcf\x2b\xb6\xe6\x02\x04\x00\x00\xff\xff\x2b\x30\x85\x77\x26\x00\x00\x00")

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

	info := bindataFileInfo{name: "001_init.down.sql", size: 38, mode: os.FileMode(420), modTime: time.Unix(1630721178, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var __001_initUpSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x9c\x53\x5d\x8f\xda\x30\x10\x7c\xcf\xaf\xd8\xb7\x03\xa9\xff\x80\xa7\xc0\x99\x53\xd4\x90\x9c\x82\x91\x38\x55\x95\xb5\x89\x17\xce\x3a\x0b\x47\xb6\x43\x8f\x56\xfd\xef\x55\x12\x8e\xaf\x10\x40\x7d\xdd\x99\x1d\xaf\x77\x67\x26\x19\x0b\x39\x03\x1e\x8e\x63\x06\xd1\x14\x92\x94\x03\x5b\x46\x73\x3e\x07\xac\x0a\xaf\xcc\xc6\xc1\x20\x00\x00\x50\x12\x3c\x7d\x7a\x78\xcd\xa2\x59\x98\xbd\xc1\x77\xf6\xf6\xad\x01\x72\xf4\xc5\xbb\xf8\x82\x6b\x81\x64\x11\xc7\x2d\x26\x09\xb5\x70\xea\x37\x41\xae\xd6\x6a\x73\x15\x96\x95\xc5\xfa\xa1\x1b\x14\x4b\xa5\x56\x45\xcb\xea\xa1\x6c\xc9\xaa\x95\x22\x09\xb9\x31\x9a\x70\x73\xc1\x59\x29\x2d\xa8\x34\xc5\xbb\x90\x84\x52\xab\x4d\xcf\x44\xf4\x59\xe8\x4a\x92\x14\xce\x1b\x8b\x6b\x12\xa5\x35\x5b\x25\xc9\xba\xe6\x7b\x3f\x7e\x1e\xf8\xf0\xcc\xa6\xe1\x22\xe6\xf0\xf4\xe7\xef\x53\xdb\x5c\xe2\x4e\x1b\x94\xa2\xb8\xbe\x8c\x02\xad\xa8\xac\x3e\x87\x8e\x32\x4f\x47\x96\x2a\x57\xae\xab\xd2\x4f\x45\x29\x1f\x1a\xd0\x79\xf4\x95\xbb\x36\x1b\x59\x6b\xac\x28\xb0\x72\x74\xe7\xd1\xdb\xe7\x72\x1e\xad\x27\x29\xd0\x03\x8f\x66\x6c\xce\xc3\xd9\x6b\x57\x6b\xb2\xc8\x32\x96\x70\x71\xa0\xb4\xcd\x55\x29\xf1\x3f\x9a\x9b\xde\xe1\x28\xd8\x7b\x39\x4a\x9e\xd9\xb2\xc7\xcb\xe2\x8a\x0f\xd2\xe4\x00\x0f\xba\xf0\x83\xba\xfb\xcd\x9e\x6a\xb5\xa5\xe1\x28\x08\x82\x1b\x29\xcb\x95\xbc\x9b\xb0\xbd\x64\x9d\x31\xce\x96\xfc\x62\xe5\xbf\x50\x6b\xf2\x8d\x07\x84\x53\x6b\xc8\x77\x9e\xb0\x73\x96\x73\x37\xf7\x48\xe5\x4a\xf6\x83\xe8\x3e\x44\x69\x55\x41\x30\x8e\x5e\xa2\xe4\x12\xfe\x0a\xa0\xb8\xc3\x6b\x1c\xd2\x2e\xf9\x3a\x61\x85\xce\x0b\x4b\xde\x2a\xda\xa2\x86\x71\x9a\xc6\x2c\x4c\xba\x3e\x98\x86\xf1\x9c\xb5\x2d\x96\x0a\x52\xdb\x3e\xe7\xec\xd7\x64\x36\x67\xf0\x3e\xb2\xd6\x94\xc6\xa1\x3e\xa4\xad\x5b\x16\x92\xb4\xda\x92\xbd\x90\xef\x27\xee\x44\x93\xa6\x13\xb9\x49\x9a\xcc\x79\x16\xd6\x7f\x5d\x7d\x88\x93\x73\x4e\xd3\x8c\x45\x2f\x49\x7d\xea\xc1\xb1\x3c\x84\x8c\x4d\x59\xc6\x92\x09\x3b\x5a\x6c\xa0\xe4\xf0\x01\xa7\xd7\x7e\x3a\x7d\x21\x4d\x9a\xd2\xa9\xfa\x28\xf8\x17\x00\x00\xff\xff\xa7\x9e\x55\x6c\xf6\x05\x00\x00")

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

	info := bindataFileInfo{name: "001_init.up.sql", size: 1526, mode: os.FileMode(420), modTime: time.Unix(1630721178, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var __002_bids_support_calculating_ratesDownSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x72\xf4\x09\x71\x0d\x52\x08\x71\x74\xf2\x71\x55\x48\xca\x4c\x29\x56\x70\x09\xf2\x0f\x50\x70\xf6\xf7\x09\xf5\xf5\x53\x48\x49\x4d\xcc\x89\x4f\xce\xcf\x4b\xcb\x2c\xca\x4d\x4d\x89\x4f\x2c\xb1\xe6\x02\x4b\x7b\xfa\xb9\xb8\x46\x28\x78\xba\x29\xb8\x46\x78\x06\x87\x04\x83\x35\xc6\x17\xa5\x26\xa7\x66\x96\x41\x95\x01\x02\x00\x00\xff\xff\xc6\x11\xea\x17\x57\x00\x00\x00")

func _002_bids_support_calculating_ratesDownSqlBytes() ([]byte, error) {
	return bindataRead(
		__002_bids_support_calculating_ratesDownSql,
		"002_bids_support_calculating_rates.down.sql",
	)
}

func _002_bids_support_calculating_ratesDownSql() (*asset, error) {
	bytes, err := _002_bids_support_calculating_ratesDownSqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "002_bids_support_calculating_rates.down.sql", size: 87, mode: os.FileMode(420), modTime: time.Unix(1632405209, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var __002_bids_support_calculating_ratesUpSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x72\xf4\x09\x71\x0d\x52\x08\x71\x74\xf2\x71\x55\x48\xca\x4c\x29\x56\x70\x74\x71\x51\x70\xf6\xf7\x09\xf5\xf5\x53\x48\x49\x4d\xcc\x89\x4f\xce\xcf\x4b\xcb\x2c\xca\x4d\x4d\x89\x4f\x2c\x51\x08\xf1\xf4\x75\x0d\x0e\x71\xf4\x0d\xb0\xe6\x72\x0e\x72\x75\x0c\x71\x55\xf0\xf4\x73\x71\x8d\x50\xf0\x74\x53\xf0\xf3\x0f\x51\x70\x8d\xf0\x0c\x0e\x09\x06\x9b\x13\x5f\x94\x9a\x9c\x9a\x59\x06\xd1\xe6\xef\x07\x16\xd3\x40\x12\xd3\xb4\xe6\x02\x04\x00\x00\xff\xff\x12\x02\x30\xf1\x7b\x00\x00\x00")

func _002_bids_support_calculating_ratesUpSqlBytes() ([]byte, error) {
	return bindataRead(
		__002_bids_support_calculating_ratesUpSql,
		"002_bids_support_calculating_rates.up.sql",
	)
}

func _002_bids_support_calculating_ratesUpSql() (*asset, error) {
	bytes, err := _002_bids_support_calculating_ratesUpSqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "002_bids_support_calculating_rates.up.sql", size: 123, mode: os.FileMode(420), modTime: time.Unix(1631305441, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var __003_auctions_add_client_addressDownSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x72\xf4\x09\x71\x0d\x52\x08\x71\x74\xf2\x71\x55\x48\x2c\x4d\x2e\xc9\xcc\xcf\x2b\x56\x70\x09\xf2\x0f\x50\x70\xf6\xf7\x09\xf5\xf5\x53\x48\xce\xc9\x4c\xcd\x2b\x89\x4f\x4c\x49\x29\x4a\x2d\x2e\xb6\xe6\x02\x04\x00\x00\xff\xff\x9e\xb9\x90\xbb\x31\x00\x00\x00")

func _003_auctions_add_client_addressDownSqlBytes() ([]byte, error) {
	return bindataRead(
		__003_auctions_add_client_addressDownSql,
		"003_auctions_add_client_address.down.sql",
	)
}

func _003_auctions_add_client_addressDownSql() (*asset, error) {
	bytes, err := _003_auctions_add_client_addressDownSqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "003_auctions_add_client_address.down.sql", size: 49, mode: os.FileMode(420), modTime: time.Unix(1632405209, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var __003_auctions_add_client_addressUpSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x72\x72\x75\xf7\xf4\xb3\xe6\x72\xf4\x09\x71\x0d\x52\x08\x71\x74\xf2\x71\x55\x48\x2c\x4d\x2e\xc9\xcc\xcf\x2b\x56\x70\x74\x71\x51\x70\xf6\xf7\x09\xf5\xf5\x53\x48\xce\xc9\x4c\xcd\x2b\x89\x4f\x4c\x49\x29\x4a\x2d\x2e\x56\x08\x71\x8d\x08\xb1\xe6\x0a\x0d\x70\x71\x0c\x41\x52\x1f\xec\x1a\x82\xae\xd0\x56\x41\x3d\xcd\xd0\xc4\xa4\x2a\xb5\xc0\x24\x3d\xb3\x24\xcb\xdc\xb8\xa8\xa8\x34\x2b\xc5\x38\xab\xdc\x2c\xb3\xa0\x28\x27\x2b\x33\xb9\xc2\xac\x2c\xc7\xa4\x3c\x29\x35\xb1\x2c\x53\x1d\x97\x33\xc0\x82\xd8\x1d\x02\xb2\xd2\xcf\x3f\x44\xc1\x2f\xd4\xc7\xc7\x9a\xcb\xd9\xdf\xd7\xd7\x33\xc4\x9a\x0b\x10\x00\x00\xff\xff\xa4\x28\x63\x82\xd5\x00\x00\x00")

func _003_auctions_add_client_addressUpSqlBytes() ([]byte, error) {
	return bindataRead(
		__003_auctions_add_client_addressUpSql,
		"003_auctions_add_client_address.up.sql",
	)
}

func _003_auctions_add_client_addressUpSql() (*asset, error) {
	bytes, err := _003_auctions_add_client_addressUpSqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "003_auctions_add_client_address.up.sql", size: 213, mode: os.FileMode(420), modTime: time.Unix(1632405209, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var __004_bids_add_won_reasonDownSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x72\xf4\x09\x71\x0d\x52\x08\x71\x74\xf2\x71\x55\x48\xca\x4c\x29\x56\x70\x09\xf2\x0f\x50\x70\xf6\xf7\x09\xf5\xf5\x53\x28\xcf\xcf\x8b\x2f\x4a\x4d\x2c\xce\xcf\xb3\xe6\xe2\x02\x04\x00\x00\xff\xff\xe1\xf7\x62\x2a\x2a\x00\x00\x00")

func _004_bids_add_won_reasonDownSqlBytes() ([]byte, error) {
	return bindataRead(
		__004_bids_add_won_reasonDownSql,
		"004_bids_add_won_reason.down.sql",
	)
}

func _004_bids_add_won_reasonDownSql() (*asset, error) {
	bytes, err := _004_bids_add_won_reasonDownSqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "004_bids_add_won_reason.down.sql", size: 42, mode: os.FileMode(420), modTime: time.Unix(1632924833, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var __004_bids_add_won_reasonUpSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x72\xf4\x09\x71\x0d\x52\x08\x71\x74\xf2\x71\x55\x48\xca\x4c\x29\x56\x70\x74\x71\x51\x70\xf6\xf7\x09\xf5\xf5\x53\x28\xcf\xcf\x8b\x2f\x4a\x4d\x2c\xce\xcf\x53\x08\x71\x8d\x08\xb1\xe6\x02\x04\x00\x00\xff\xff\x99\x5a\x93\xde\x2d\x00\x00\x00")

func _004_bids_add_won_reasonUpSqlBytes() ([]byte, error) {
	return bindataRead(
		__004_bids_add_won_reasonUpSql,
		"004_bids_add_won_reason.up.sql",
	)
}

func _004_bids_add_won_reasonUpSql() (*asset, error) {
	bytes, err := _004_bids_add_won_reasonUpSqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "004_bids_add_won_reason.up.sql", size: 45, mode: os.FileMode(420), modTime: time.Unix(1632929210, 0)}
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
	"001_init.down.sql":                           _001_initDownSql,
	"001_init.up.sql":                             _001_initUpSql,
	"002_bids_support_calculating_rates.down.sql": _002_bids_support_calculating_ratesDownSql,
	"002_bids_support_calculating_rates.up.sql":   _002_bids_support_calculating_ratesUpSql,
	"003_auctions_add_client_address.down.sql":    _003_auctions_add_client_addressDownSql,
	"003_auctions_add_client_address.up.sql":      _003_auctions_add_client_addressUpSql,
	"004_bids_add_won_reason.down.sql":            _004_bids_add_won_reasonDownSql,
	"004_bids_add_won_reason.up.sql":              _004_bids_add_won_reasonUpSql,
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
	"001_init.down.sql":                           &bintree{_001_initDownSql, map[string]*bintree{}},
	"001_init.up.sql":                             &bintree{_001_initUpSql, map[string]*bintree{}},
	"002_bids_support_calculating_rates.down.sql": &bintree{_002_bids_support_calculating_ratesDownSql, map[string]*bintree{}},
	"002_bids_support_calculating_rates.up.sql":   &bintree{_002_bids_support_calculating_ratesUpSql, map[string]*bintree{}},
	"003_auctions_add_client_address.down.sql":    &bintree{_003_auctions_add_client_addressDownSql, map[string]*bintree{}},
	"003_auctions_add_client_address.up.sql":      &bintree{_003_auctions_add_client_addressUpSql, map[string]*bintree{}},
	"004_bids_add_won_reason.down.sql":            &bintree{_004_bids_add_won_reasonDownSql, map[string]*bintree{}},
	"004_bids_add_won_reason.up.sql":              &bintree{_004_bids_add_won_reasonUpSql, map[string]*bintree{}},
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
