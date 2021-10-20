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
// migrations/005_auctions_add_providers.down.sql
// migrations/005_auctions_add_providers.up.sql
// migrations/006_consume_bidbot_events.down.sql
// migrations/006_consume_bidbot_events.up.sql
// migrations/007_bids_add_deal_failed_at.down.sql
// migrations/007_bids_add_deal_failed_at.up.sql
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

	info := bindataFileInfo{name: "001_init.down.sql", size: 38, mode: os.FileMode(436), modTime: time.Unix(1630327856, 0)}
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

	info := bindataFileInfo{name: "001_init.up.sql", size: 1526, mode: os.FileMode(436), modTime: time.Unix(1630327856, 0)}
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

	info := bindataFileInfo{name: "002_bids_support_calculating_rates.down.sql", size: 87, mode: os.FileMode(436), modTime: time.Unix(1633021059, 0)}
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

	info := bindataFileInfo{name: "002_bids_support_calculating_rates.up.sql", size: 123, mode: os.FileMode(436), modTime: time.Unix(1631615050, 0)}
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

	info := bindataFileInfo{name: "003_auctions_add_client_address.down.sql", size: 49, mode: os.FileMode(436), modTime: time.Unix(1633021059, 0)}
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

	info := bindataFileInfo{name: "003_auctions_add_client_address.up.sql", size: 213, mode: os.FileMode(436), modTime: time.Unix(1633021059, 0)}
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

	info := bindataFileInfo{name: "004_bids_add_won_reason.down.sql", size: 42, mode: os.FileMode(436), modTime: time.Unix(1633021059, 0)}
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

	info := bindataFileInfo{name: "004_bids_add_won_reason.up.sql", size: 45, mode: os.FileMode(436), modTime: time.Unix(1633021059, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var __005_auctions_add_providersDownSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x72\xf4\x09\x71\x0d\x52\x08\x71\x74\xf2\x71\x55\x48\x2c\x4d\x2e\xc9\xcc\xcf\x2b\x56\x70\x09\xf2\x0f\x50\x70\xf6\xf7\x09\xf5\xf5\x53\x28\x28\xca\x2f\xcb\x4c\x49\x2d\x2a\xb6\xe6\x02\x04\x00\x00\xff\xff\x35\xb4\x2f\x33\x2c\x00\x00\x00")

func _005_auctions_add_providersDownSqlBytes() ([]byte, error) {
	return bindataRead(
		__005_auctions_add_providersDownSql,
		"005_auctions_add_providers.down.sql",
	)
}

func _005_auctions_add_providersDownSql() (*asset, error) {
	bytes, err := _005_auctions_add_providersDownSqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "005_auctions_add_providers.down.sql", size: 44, mode: os.FileMode(436), modTime: time.Unix(1633453830, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var __005_auctions_add_providersUpSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x72\xf4\x09\x71\x0d\x52\x08\x71\x74\xf2\x71\x55\x48\x2c\x4d\x2e\xc9\xcc\xcf\x2b\x56\x70\x74\x71\x51\x70\xf6\xf7\x09\xf5\xf5\x53\x28\x28\xca\x2f\xcb\x4c\x49\x2d\x2a\x56\x08\x71\x8d\x08\x89\x8e\xb5\xe6\x02\x04\x00\x00\xff\xff\xb7\x08\x84\xce\x32\x00\x00\x00")

func _005_auctions_add_providersUpSqlBytes() ([]byte, error) {
	return bindataRead(
		__005_auctions_add_providersUpSql,
		"005_auctions_add_providers.up.sql",
	)
}

func _005_auctions_add_providersUpSql() (*asset, error) {
	bytes, err := _005_auctions_add_providersUpSqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "005_auctions_add_providers.up.sql", size: 50, mode: os.FileMode(436), modTime: time.Unix(1633453830, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var __006_consume_bidbot_eventsDownSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x72\x09\xf2\x0f\x50\x08\x71\x74\xf2\x71\x55\xf0\x74\x53\x70\x8d\xf0\x0c\x0e\x09\x56\x48\xca\x4c\x89\x4f\x2d\x4b\xcd\x2b\x29\xb6\xe6\x02\x2b\x70\xf5\x0b\xf5\x45\x88\xc6\x97\x54\x16\xa4\x42\x65\xd0\xb5\x16\x97\xe4\x17\x25\xa6\xa7\xc6\x17\x14\xe5\x97\x65\xa6\xa4\x16\x15\x5b\x73\x01\x02\x00\x00\xff\xff\x4a\x5a\x98\x47\x63\x00\x00\x00")

func _006_consume_bidbot_eventsDownSqlBytes() ([]byte, error) {
	return bindataRead(
		__006_consume_bidbot_eventsDownSql,
		"006_consume_bidbot_events.down.sql",
	)
}

func _006_consume_bidbot_eventsDownSql() (*asset, error) {
	bytes, err := _006_consume_bidbot_eventsDownSqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "006_consume_bidbot_events.down.sql", size: 99, mode: os.FileMode(436), modTime: time.Unix(1633703350, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var __006_consume_bidbot_eventsUpSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xa4\x92\xc1\x6e\xe2\x30\x10\x86\xef\x79\x8a\xb9\x01\x52\xdf\xa0\xa7\x2c\x35\xab\x68\xc1\xa0\xc4\x48\xe5\x64\x99\x78\x12\x46\x1b\xec\xc8\x9e\xa6\xcb\x3e\xfd\x8a\x84\xa0\xa6\x6a\xb9\xec\xd1\xbf\xbf\xf9\x35\xf3\xcf\x2c\x73\x91\x2a\x01\x2a\xfd\xb1\x16\x90\xad\x40\x6e\x15\x88\xd7\xac\x50\x05\x44\xf6\xc1\xd4\xa8\xdb\xe0\x3b\xb2\x18\x22\xcc\x13\x00\x00\xb2\xc0\xf8\x87\x61\x97\x67\x9b\x34\x3f\xc0\x2f\x71\x78\xea\x3f\x8e\x64\x8f\x9e\x75\x87\x21\x92\x77\x03\x74\xf5\x93\xfb\xf5\x7a\x20\x2c\x9a\x46\x47\x36\x81\xf5\x3b\x39\xeb\xdf\xe1\x48\x35\xb9\xcf\x58\x49\x56\xd7\xc1\x74\xc4\x17\x5d\x7a\x57\x51\xfd\x16\xd0\xc2\xd1\xfb\x06\x8d\x7b\x00\x47\x0e\x54\xf2\x37\x60\x45\x21\xb2\x8e\x88\x4e\x1b\x06\xa6\x33\x46\x36\xe7\xf6\x4e\xc1\x8b\x58\xa5\xfb\xb5\x82\xe5\x3e\xcf\x85\x54\x5a\x65\x1b\x51\xa8\x74\xb3\x1b\xea\x1b\xf3\xff\xe5\x6f\xee\x84\xa6\xe1\xd3\x65\xe2\xf1\x25\x80\x21\xf8\xd0\x87\xd8\xff\x2e\x9e\x93\x64\x5c\xd6\x61\x27\xae\x61\x6b\xec\xd0\xb1\xe6\x4b\x8b\x90\x16\x20\xe4\x7e\x03\xf3\xd9\x10\x6f\x85\x5c\x9e\xc8\xd5\xb3\x27\x98\xf5\x4e\x13\x65\x60\xe8\xdc\xfa\xc0\x23\xe4\xec\x54\xa8\xc8\x99\x86\xfe\xa2\xbd\x5b\xa0\x9d\x2d\x9e\x93\x07\x17\x73\xef\x69\x3c\x95\xab\x30\x9e\xcb\x74\x19\x1f\x5a\xff\x34\xc9\x94\x33\xcc\x78\x6e\x39\x02\x39\xbe\x55\xde\x73\x19\xde\x27\xd3\xb6\xe8\xd0\x7e\xbd\x95\x81\x09\x58\x22\x75\x8f\x99\xe5\x56\x16\x2a\x4f\x33\xa9\xa0\xfa\xad\x6f\x9d\xaf\xb6\xb9\xc8\x7e\xca\xeb\x8d\xcf\x07\x69\x01\xb9\x58\x89\x5c\xc8\xa5\xe8\x07\x8e\x73\xb2\x8b\x71\x43\xb7\x6c\x32\xf9\x22\x5e\xbf\xcd\x66\xf4\xde\xca\x0f\xe2\xe8\xfe\x9c\xfc\x0b\x00\x00\xff\xff\x36\xc9\x9e\xc0\x92\x03\x00\x00")

func _006_consume_bidbot_eventsUpSqlBytes() ([]byte, error) {
	return bindataRead(
		__006_consume_bidbot_eventsUpSql,
		"006_consume_bidbot_events.up.sql",
	)
}

func _006_consume_bidbot_eventsUpSql() (*asset, error) {
	bytes, err := _006_consume_bidbot_eventsUpSqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "006_consume_bidbot_events.up.sql", size: 914, mode: os.FileMode(436), modTime: time.Unix(1633703350, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var __007_bids_add_deal_failed_atDownSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x72\xf4\x09\x71\x0d\x52\x08\x71\x74\xf2\x71\x55\x48\xca\x4c\x29\x56\x70\x09\xf2\x0f\x50\x70\xf6\xf7\x09\xf5\xf5\x53\x48\x49\x4d\xcc\x89\x4f\x4b\xcc\xcc\x49\x4d\x89\x4f\x2c\xb1\xe6\xe2\x02\x04\x00\x00\xff\xff\x61\x34\xf9\x26\x2e\x00\x00\x00")

func _007_bids_add_deal_failed_atDownSqlBytes() ([]byte, error) {
	return bindataRead(
		__007_bids_add_deal_failed_atDownSql,
		"007_bids_add_deal_failed_at.down.sql",
	)
}

func _007_bids_add_deal_failed_atDownSql() (*asset, error) {
	bytes, err := _007_bids_add_deal_failed_atDownSqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "007_bids_add_deal_failed_at.down.sql", size: 46, mode: os.FileMode(436), modTime: time.Unix(1633957075, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var __007_bids_add_deal_failed_atUpSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x72\xf4\x09\x71\x0d\x52\x08\x71\x74\xf2\x71\x55\x48\xca\x4c\x29\x56\x70\x74\x71\x51\x70\xf6\xf7\x09\xf5\xf5\x53\x48\x49\x4d\xcc\x89\x4f\x4b\xcc\xcc\x49\x4d\x89\x4f\x2c\x51\x08\xf1\xf4\x75\x0d\x0e\x71\xf4\x0d\xb0\xe6\xe2\x02\x04\x00\x00\xff\xff\x43\xc8\x19\x2c\x37\x00\x00\x00")

func _007_bids_add_deal_failed_atUpSqlBytes() ([]byte, error) {
	return bindataRead(
		__007_bids_add_deal_failed_atUpSql,
		"007_bids_add_deal_failed_at.up.sql",
	)
}

func _007_bids_add_deal_failed_atUpSql() (*asset, error) {
	bytes, err := _007_bids_add_deal_failed_atUpSqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "007_bids_add_deal_failed_at.up.sql", size: 55, mode: os.FileMode(436), modTime: time.Unix(1633957075, 0)}
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
	"005_auctions_add_providers.down.sql":         _005_auctions_add_providersDownSql,
	"005_auctions_add_providers.up.sql":           _005_auctions_add_providersUpSql,
	"006_consume_bidbot_events.down.sql":          _006_consume_bidbot_eventsDownSql,
	"006_consume_bidbot_events.up.sql":            _006_consume_bidbot_eventsUpSql,
	"007_bids_add_deal_failed_at.down.sql":        _007_bids_add_deal_failed_atDownSql,
	"007_bids_add_deal_failed_at.up.sql":          _007_bids_add_deal_failed_atUpSql,
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
	"005_auctions_add_providers.down.sql":         &bintree{_005_auctions_add_providersDownSql, map[string]*bintree{}},
	"005_auctions_add_providers.up.sql":           &bintree{_005_auctions_add_providersUpSql, map[string]*bintree{}},
	"006_consume_bidbot_events.down.sql":          &bintree{_006_consume_bidbot_eventsDownSql, map[string]*bintree{}},
	"006_consume_bidbot_events.up.sql":            &bintree{_006_consume_bidbot_eventsUpSql, map[string]*bintree{}},
	"007_bids_add_deal_failed_at.down.sql":        &bintree{_007_bids_add_deal_failed_atDownSql, map[string]*bintree{}},
	"007_bids_add_deal_failed_at.up.sql":          &bintree{_007_bids_add_deal_failed_atUpSql, map[string]*bintree{}},
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
