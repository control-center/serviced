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
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/control-center/serviced/commons/diet"
	"github.com/control-center/serviced/utils"
	"github.com/zenoss/glog"
)

var storageStatsUpdateInterval time.Duration = 300

// SetStorageStatsUpdateInterval sets the interval to update storage stats at
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

func getMetadataBlock(pool string) (string, error) {
	cmd := exec.Command("dmsetup", "status", pool)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	parts := strings.Fields(string(out))
	return strings.TrimSpace(parts[6]), nil
}

// reserveMetadataSnap reserves a userspace snapshot of a thin pool's metadata.
// We shell out to send the message to the device-mapper subsystem, but we lock
// our internal DeviceSet to avoid any complications with creating or removing
// devices simultaneously.
func (d *DeviceMapperDriver) reserveMetadataSnap(pool string) error {
	d.DeviceSet.Lock()
	defer d.DeviceSet.Unlock()
	cmd := exec.Command("dmsetup", "message", pool, "0", "reserve_metadata_snap")
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

// releaseMetadataSnap releases an existing userspace snapshot of a thin pool's
// metadata.  We shell out to send the message to the device-mapper subsystem,
// but we lock our internal DeviceSet to avoid any complications with creating
// or removing devices simultaneously.
func (d *DeviceMapperDriver) releaseMetadataSnap(pool string) error {
	d.DeviceSet.Lock()
	defer d.DeviceSet.Unlock()
	cmd := exec.Command("dmsetup", "message", pool, "0", "release_metadata_snap")
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

// getDevices acts on an existing userspace metadata snapshot, and has no
// potential devicemapper conflicts. It simply dumps the XML and parses it.
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

// getDeviceSize calls lsblk in a subprocess to retrieve the size in bytes of
// a given device.
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

// dfStats contains stats reported by df
type dfStats struct {
	BlockSize uint64
	FilesystemPath string
	BlocksTotal uint64
	BlocksUsed uint64
	BlocksAvailable uint64
	MountPoint string
}

// filesystemNotMountedErr is the error raised when a filesystem is not mounted
var filesystemNotMountedErr = errors.New("Filesystem not mounted.")

// getFilesystemStats calls df to see if the filesystem is mounted and
// gets the parsed info for it if it is.
// If it is not mounted, it calls getUnmountedFilesystemStats for the info.
func getFilesystemStats(dev string) (uint64, uint64, error) {
	totalBlocks, freeBlocks, err := getMountedFilesystemStats(dev)
	// Catch the filesystemNotMountedErr and try getUnmountedFilesystemStats
	if err == filesystemNotMountedErr {
		totalBlocks, freeBlocks, err = getUnmountedFilesystemStats(dev)
		if err != nil {
			return 0, 0, err
		}
	} else if err != nil {
		return 0, 0, err
	}
	return totalBlocks, freeBlocks, err
}

// getMountedFilesystemStats calls df to get filesystem stats for a mounted dev
func getMountedFilesystemStats(dev string) (uint64, uint64, error) {
	// Specify 1 byte "blocks" for ease
	cmd := exec.Command("df", "-B 1")
	defer cmd.Wait()
	out, err := cmd.StdoutPipe()
	if err != nil {
		return 0, 0, err
	}
	defer out.Close()
	if err = cmd.Start(); err != nil {
		return 0, 0, err
	}
	mounted := false
	var totalBlocks uint64
	var freeBlocks uint64
	dfStatsSlice, err := parseDfOutput(out)
	if err != nil {
		return 0, 0, err
	}
	for _, stats := range dfStatsSlice {
		if stats.FilesystemPath == dev {
			mounted = true
			totalBlocks = stats.BlocksTotal
			freeBlocks = stats.BlocksAvailable
			break
		}
	}
	if !mounted {
		return 0, 0, filesystemNotMountedErr
	}

	return totalBlocks, freeBlocks, err
}

// parseDfOutput attempts to parse output from a df call.  It supports multiple
// options for block sizes.  Returns a dfStats object for each mounted device
// in the df ouput
func parseDfOutput(r io.Reader) ([]*dfStats, error) {
	getBytes := func(blockStr string) (uint64, error) {
		base := uint64(1024)
		bytes := uint64(0)
		b := uint64(0)
		var err error
		// in df output, kB, MB, etc are powers of 1000, K, M, etc are powers of 1024
		if strings.HasSuffix(blockStr, "B") {
			base = 1000
		}
		blockStr = strings.TrimSuffix(blockStr, "B")
		if strings.HasSuffix(blockStr, "K") {
			b, err = strconv.ParseUint(strings.TrimSuffix(blockStr, "K"), 10, 64)
			bytes = b * base
		} else if strings.HasSuffix(blockStr, "M") {
			b, err = strconv.ParseUint(strings.TrimSuffix(blockStr, "M"), 10, 64)
			bytes = b * base * base
		} else if strings.HasSuffix(blockStr, "G") {
			b, err = strconv.ParseUint(strings.TrimSuffix(blockStr, "G"), 10, 64)
			bytes = b * base * base * base
		} else if strings.HasSuffix(blockStr, "T") {
			b, err = strconv.ParseUint(strings.TrimSuffix(blockStr, "T"), 10, 64)
			bytes = b * base * base * base * base
		} else {
			bytes, err =  strconv.ParseUint(blockStr, 10, 64)
		}

		return bytes, err
	}
	scanner := bufio.NewScanner(r)
	statsSlice := make([]*dfStats, 0, 3)
	var err error
	var blockSize uint64
	for scanner.Scan() {
		line := scanner.Text()
		f := strings.Fields(line)
		// Read block size from output header
		if strings.HasPrefix(f[0], "Filesystem") {
			blockSizeStr := strings.Split(f[1], "-")[0]
			blockSize, err = getBytes(blockSizeStr)
			if err != nil {
				return statsSlice, err
			}
		} else {
			filesystemPath := f[0]
			totalBlocks, err := strconv.ParseUint(f[1], 10, 64)
			if err != nil {
				return statsSlice, err
			}
			usedBlocks, err := strconv.ParseUint(f[2], 10, 64)
			if err != nil {
				return statsSlice, err
			}
			availableBlocks, err := strconv.ParseUint(f[3], 10, 64)
			if err != nil {
				return statsSlice, err
			}
			mountPoint := f[5]
			statsSlice = append(statsSlice, &dfStats{
				BlockSize: blockSize,
				FilesystemPath: filesystemPath,
				BlocksTotal: totalBlocks,
				BlocksUsed: usedBlocks,
				BlocksAvailable: availableBlocks,
				MountPoint: mountPoint,
			})
		}
	}
	return statsSlice, nil
}

// filesystemStats contains stats for a filesystem
type filesystemStats struct {
	BlocksTotal     uint64
	BlockSize       uint64
	FreeBlocks      uint64
	UnusableBlocks  uint64
	Superblocks     uint64
	GroupsTotal     uint64
	JournalLength   uint64
	SuperblockSize  uint64
	GroupDescSize   uint64
	BitmapSize      uint64
	InodeBitmapSize uint64
	InodeTableSize  uint64
}

// getUnmountedFilesystemStats uses a dumpe2fs parser to find filesystem information.
// We get a lot of info from dumpe2fs to calculate the usable space on the
// filesystem, as this is what df shows, and we want consistancy since
// df is used when the disk is mounted (df is more up-to-date).
func getUnmountedFilesystemStats(dev string) (uint64, uint64, error) {
	cmd := exec.Command("dumpe2fs", dev)
	defer cmd.Wait()
	out, err := cmd.StdoutPipe()
	if err != nil {
		return 0, 0, err
	}
	defer out.Close()

	if err = cmd.Start(); err != nil {
		return 0, 0, err
	}

	fs, err := parseDumpe2fsOutput(out)
	if err != nil {
		return 0, 0, err
	}

	return (fs.BlocksTotal - fs.UnusableBlocks) * fs.BlockSize,
		fs.FreeBlocks * fs.BlockSize,
		nil
}

// parseDumpe2fsOutput reads output from a dumpe2fs call to
// create a filesystemStats object
func parseDumpe2fsOutput(r io.Reader) (*filesystemStats, error) {
	// Custom bufio.Split() function to split tokens by either comma or new line
	atCommaOrNewLine := func(data []byte, atEOF bool) (int, []byte, error) {
		advance := 0
		var token []byte

		for _, b := range data {
			// consume the comma or newline by advancing passed it
			if string(b) == "," || string(b) == "\n" {
				advance++
				return advance, token, nil
			}
			token = append(token, b)
			advance++
		}
		return 0, nil, nil
	}

	malformedRangeErr := errors.New("Malformed range.  Expected form: \"x-y\"")

	// Gets the inclusive difference of a range
	// e.g. rangeDiffIncl("1-15") == 15
	rangeDiffIncl := func(rangeStr string) (uint64, error) {
		splitRange := strings.Split(rangeStr, "-")
		if len(splitRange) != 2 {
			return 0, malformedRangeErr
		}
		uLeft, err := strconv.ParseUint(splitRange[0], 10, 64)
		if err != nil {
			return 0, err
		}
		uRight, err := strconv.ParseUint(splitRange[1], 10, 64)
		if err != nil {
			return 0, err
		}
		return uRight - uLeft + 1, nil
	}

	// Returns number of blocks consumed from string that contains block locations
	getBlocksConsumed := func(blockString string) (uint64, error) {
		// If it's a range, get the span, otherwise it only takes up one block
		if strings.Contains(blockString, "-") {
			return rangeDiffIncl(blockString)
		}
		_, err := strconv.ParseUint(blockString, 10, 64)
		if err != nil {
			return 0, err
		}
		return 1, nil
	}

	var err error
	fs := &filesystemStats{}
	scanner := bufio.NewScanner(r)
	scanner.Split(atCommaOrNewLine)

	for scanner.Scan() {
		t := strings.TrimSpace(scanner.Text())
		// We need to count the groups and superblocks, so we know how much
		// usable space the fs has (this will keep size consistant with df results)
		// We're counting the groups by the Checksum prefix because 'Group' is used
		// in other places and it's cheaper than a regexp
		if strings.HasPrefix(t, "Checksum") {
			fs.GroupsTotal++
		} else if strings.Contains(t, "superblock") {
			fs.Superblocks++
		}

		// These are described by dumpe2fs like:
		// "Block bitmap at x" or "Group descriptors at x-y"
		if s := strings.Split(t, " at "); len(s) == 2 {
			loc := strings.Split(s[1], " ")[0]
			if fs.SuperblockSize == 0 && s[0] == "Primary superblock" {
				fs.SuperblockSize, err = getBlocksConsumed(loc)
			} else if fs.GroupDescSize == 0 && s[0] == "Group descriptors" {
				fs.GroupDescSize, err = getBlocksConsumed(loc)
			} else if fs.BitmapSize == 0 && s[0] == "Block bitmap" {
				fs.BitmapSize, err = getBlocksConsumed(loc)
			} else if fs.InodeBitmapSize == 0 && s[0] == "Inode bitmap" {
				fs.InodeBitmapSize, err = getBlocksConsumed(loc)
			} else if fs.InodeTableSize == 0 && s[0] == "Inode table" {
				fs.InodeTableSize, err = getBlocksConsumed(loc)
			}
		} else {
			f := strings.Fields(t)
			if fs.BlocksTotal == 0 && strings.HasPrefix(t, "Block count:") {
				fs.BlocksTotal, err = strconv.ParseUint(f[2], 10, 64)
			} else if fs.FreeBlocks == 0 && strings.HasPrefix(t, "Free blocks:") {
				fs.FreeBlocks, err = strconv.ParseUint(f[2], 10, 64)
			} else if fs.BlockSize == 0 && strings.HasPrefix(t, "Block size:") {
				fs.BlockSize, err = strconv.ParseUint(f[2], 10, 64)
			} else if fs.JournalLength == 0 && strings.HasPrefix(t, "Journal length:") {
				fs.JournalLength, err = strconv.ParseUint(f[2], 10, 64)
			}
		}
		if err != nil {
			return nil, err
		}
	}
	sizePerSuperblockGroup := fs.SuperblockSize + fs.GroupDescSize + fs.BitmapSize +
		fs.InodeBitmapSize + fs.InodeTableSize
	sizePerNormalGroup := fs.BitmapSize + fs.InodeBitmapSize + fs.InodeTableSize
	// fs.BlocksTotal - fs.UnusableBlocks should be the size shown by df
	fs.UnusableBlocks = (sizePerSuperblockGroup * fs.Superblocks) +
		(sizePerNormalGroup * (fs.GroupsTotal - fs.Superblocks)) +
		fs.JournalLength
	return fs, nil
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

func (d *DeviceMapperDriver) getDeviceBlockStats(pool, metadatadev string) (map[int]*DeviceBlockStats, error) {
	var value map[int]*DeviceBlockStats
	pool = fmt.Sprintf("/dev/mapper/%s", strings.TrimPrefix(pool, "/dev/mapper/"))
	statscache.locks.LockKey(pool)
	defer statscache.locks.UnlockKey(pool)
	val, ok := statscache.cache[pool]
	if !ok || time.Now().After(val.expiry) {
		glog.Infof("Refreshing storage stats cache from thin pool metadata")
		// Check for an existing snapshot of the metadata device.
		hasSnap, err := hasMetadataSnap(pool)
		if err != nil {
			return nil, err
		}
		// TODO: Implement file locking to avoid stomping on metadata snaps by out
		// of band processes
		if hasSnap {
			// Release the existing snapshot.
			if err := d.releaseMetadataSnap(pool); err != nil {
				return nil, err
			}
		}
		// Take a userspace-accessible snapshot of the metadata device.
		if err := d.reserveMetadataSnap(pool); err != nil {
			return nil, err
		}
		defer d.releaseMetadataSnap(pool)
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
		// Don't cache if there aren't any devices besides the base device yet
		// (e.g., initial deployment)
		if len(value) > 1 {
			statscache.cache[pool] = &cachedStat{
				expiry: time.Now().Add(storageStatsUpdateInterval * time.Second),
				pool:   pool,
				value:  value,
			}
		}
	} else {
		value = val.value
	}
	return value, nil
}
