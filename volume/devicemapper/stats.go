// Copyright 2016 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// +build linux,!darwin

package devicemapper

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/control-center/serviced/commons/diet"
	"github.com/control-center/serviced/utils"
)

var storageStatsUpdateInterval time.Duration = 300

func SetStorageStatsUpdateInterval(interval int) {
	storageStatsUpdateInterval = time.Duration(interval)
}

// DeviceBlockStats represents a devicemapper device
type DeviceBlockStats struct {
	DeviceID int
	diet     *diet.Diet
}

// RangeMapping represents a range of blocks in a metadata xml file
type RangeMapping struct {
	OriginBegin int `xml:"origin_begin,attr"`
	DataBegin   int `xml:"data_begin,attr"`
	Length      int `xml:"length,attr"`
	Time        int `xml:"time,attr"`
}

// SingleMapping represents a single block in a metadata xml file
type SingleMapping struct {
	OriginBlock int `xml:"origin_block,attr"`
	DataBlock   int `xml:"data_block,attr"`
	Time        int `xml:"time,attr"`
}

// UniqueBlocks returns the number of unique blocks that this device has
// allocated compared to another device
func (d *DeviceBlockStats) UniqueBlocks(other *DeviceBlockStats) uint64 {
	return d.diet.Total() - d.diet.IntersectionAll(other.diet)
}

func hasMetadataSnap(pool string) (bool, error) {
	block, err := getMetadataBlock(pool)
	if err != nil {
		return false, err
	}
	return block != "-", nil
}

func reserveMetadataSnap(pool string) error {
	cmd := exec.Command("dmsetup", "message", pool, "0", "reserve_metadata_snap")
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func releaseMetadataSnap(pool string) error {
	cmd := exec.Command("dmsetup", "message", pool, "0", "release_metadata_snap")
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func getMetadataBlock(pool string) (string, error) {
	cmd := exec.Command("dmsetup", "status", pool)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	parts := strings.Fields(string(out))
	return strings.TrimSpace(parts[6]), nil
}

func getDevices(pool, block, metadatadev string) (map[int]*DeviceBlockStats, error) {
	cmd := exec.Command("thin_dump", "-f", "xml", metadatadev, "-m", block)
	stdout, err := cmd.StdoutPipe()
	cmd.Stderr = os.Stderr
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	devices, err := parseMetadata(stdout)
	if err != nil {
		return nil, err
	}
	if err := cmd.Wait(); err != nil {
		return nil, err
	}
	return devices, nil
}

func parseMetadata(r io.Reader) (map[int]*DeviceBlockStats, error) {
	devices := make(map[int]*DeviceBlockStats)
	decoder := xml.NewDecoder(r)
	var current *DeviceBlockStats
	for {
		t, err := decoder.Token()
		if t == nil || err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		switch se := t.(type) {
		case xml.EndElement:
			if se.Name.Local == "device" {
				if current == nil {
					// current should never be nil in this scenario, but if it
					// is, let's assume there's a good reason and just move on
					continue
				}
				// Balance the tree (necessary for performance, since the data
				// is ordered when we get it, so the tree is pretty much just
				// a linked list at this point)
				current.diet.Balance()
				devices[current.DeviceID], current = current, nil
			}
		case xml.StartElement:
			if se.Name.Local == "device" {
				for _, attr := range se.Attr {
					if attr.Name.Local == "dev_id" {
						id, err := strconv.Atoi(attr.Value)
						if err != nil {
							return nil, err
						}
						current = &DeviceBlockStats{id, diet.NewDiet()}
						break
					}
				}
				continue
			}
			if se.Name.Local == "range_mapping" {
				var m RangeMapping
				if err := decoder.DecodeElement(&m, &se); err != nil {
					return nil, err
				}
				current.diet.Insert(uint64(m.DataBegin), uint64(m.DataBegin+m.Length-1))
			}
			if se.Name.Local == "single_mapping" {
				var m SingleMapping
				if err := decoder.DecodeElement(&m, &se); err != nil {
					return nil, err
				}
				current.diet.Insert(uint64(m.DataBlock), uint64(m.DataBlock))
			}
		}
	}
	return devices, nil
}

func getDeviceSize(dev string) (uint64, error) {
	cmd := exec.Command("lsblk", "-dbno", "SIZE", dev)
	out, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	var size uint64
	if size, err = strconv.ParseUint(strings.TrimSpace(string(out)), 10, 64); err != nil {
		return 0, err
	}
	return size, nil
}

func getFilesystemStats(dev string) (uint64, uint64, error) {
	cmd := exec.Command("dumpe2fs", dev)
	out, err := cmd.StdoutPipe()
	if err != nil {
		return 0, 0, err
	}
	if err = cmd.Start(); err != nil {
		return 0, 0, err
	}
	scanner := bufio.NewScanner(out)
	var blocksize, totalblocks, freeblocks uint64
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			break
		}
		if strings.HasPrefix(line, "Block count:") {
			n := strings.Fields(line)[2]
			if totalblocks, err = strconv.ParseUint(n, 10, 64); err != nil {
				return 0, 0, err
			}
		} else if strings.HasPrefix(line, "Free blocks:") {
			n := strings.Fields(line)[2]
			if freeblocks, err = strconv.ParseUint(n, 10, 64); err != nil {
				return 0, 0, err
			}
		} else if strings.HasPrefix(line, "Block size:") {
			n := strings.Fields(line)[2]
			if blocksize, err = strconv.ParseUint(n, 10, 64); err != nil {
				return 0, 0, err
			}
		}
	}
	return totalblocks * blocksize, freeblocks * blocksize, nil
}

var statscache = statCache{make(map[string]*cachedStat), utils.NewMutexMap()}

type statCache struct {
	cache map[string]*cachedStat
	locks *utils.MutexMap
}

type cachedStat struct {
	expiry time.Time
	pool   string
	value  map[int]*DeviceBlockStats
}

func getDeviceBlockStats(pool, metadatadev string) (map[int]*DeviceBlockStats, error) {
	var value map[int]*DeviceBlockStats
	pool = fmt.Sprintf("/dev/mapper/%s", strings.TrimPrefix(pool, "/dev/mapper/"))
	statscache.locks.LockKey(pool)
	defer statscache.locks.UnlockKey(pool)
	val, ok := statscache.cache[pool]
	if !ok || time.Now().After(val.expiry) {
		// Check for an existing snapshot of the metadata device.
		hasSnap, err := hasMetadataSnap(pool)
		if err != nil {
			return nil, err
		}
		// TODO: Implement file locking to avoid stomping on metadata snaps by out
		// of band processes
		if hasSnap {
			// Release the existing snapshot.
			if err := releaseMetadataSnap(pool); err != nil {
				return nil, err
			}
		}
		// Take a userspace-accessible snapshot of the metadata device.
		if err := reserveMetadataSnap(pool); err != nil {
			return nil, err
		}
		defer releaseMetadataSnap(pool)
		// Ask for the block at which the metadata snap is accessible
		block, err := getMetadataBlock(pool)
		if err != nil {
			return nil, err
		}
		// Dump the metadata from the snapshot to XML, parse it, and return the
		// resulting DeviceBlockStats objects.
		value, err = getDevices(pool, block, metadatadev)
		if err != nil {
			return nil, err
		}
		statscache.cache[pool] = &cachedStat{
			expiry: time.Now().Add(storageStatsUpdateInterval * time.Second),
			pool:   pool,
			value:  value,
		}
	} else {
		value = val.value
	}
	return value, nil
}
