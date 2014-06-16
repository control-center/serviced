package nfs

import (
	"reflect"
	"strings"
	"testing"
)

var testMountReader = strings.NewReader(`rootfs / rootfs rw 0 0
udev /dev devtmpfs rw,relatime,size=49455028k,nr_inodes=12363757,mode=755 0 0
none /var/lib/docker/aufs/mnt/2c563daaf020b0a3507fc8e58c7573a96572a565b6d5d27497d4af8871485c49 aufs rw,relatime,si=9f9172191782cb87 0 0
/dev/mapper/ubuntu--vg-root /exports/serviced_var ext4 rw,relatime,errors=remount-ro,data=ordered 0 0`)

var expectedMounts = []mountInstance{
	mountInstance{
		Src:       "rootfs",
		Dst:       "/",
		Type:      "rootfs",
		Options:   mountOptions{"rw": ""},
		Dump:      0,
		FsckOrder: 0},
	mountInstance{
		Src:       "udev",
		Dst:       "/dev",
		Type:      "devtmpfs",
		Options:   mountOptions{"rw": "", "relatime": "", "size": "49455028k", "nr_inodes": "12363757", "mode": "755"},
		Dump:      0,
		FsckOrder: 0},
	mountInstance{
		Src:       "none",
		Dst:       "/var/lib/docker/aufs/mnt/2c563daaf020b0a3507fc8e58c7573a96572a565b6d5d27497d4af8871485c49",
		Type:      "aufs",
		Options:   mountOptions{"rw": "", "relatime": "", "si": "9f9172191782cb87"},
		Dump:      0,
		FsckOrder: 0},
	mountInstance{
		Src:       "/dev/mapper/ubuntu--vg-root",
		Dst:       "/exports/serviced_var",
		Type:      "ext4",
		Options:   mountOptions{"rw": "", "relatime": "", "errors": "remount-ro", "data": "ordered"},
		Dump:      0,
		FsckOrder: 0},
}

func TestParseMounts(t *testing.T) {

	s, err := parseMounts(testMountReader)
	if err != nil {
		t.Fatalf("unexpected error parsing mounts: %s", err)
	}
	if s == nil || len(s) != len(expectedMounts) {
		t.Fatalf("mount count is different")
	}
	for i := range s {
		if !reflect.DeepEqual(s[i], expectedMounts[i]) {
			t.Fatalf("%d: got %v expected %v", i, s[i], expectedMounts[i])
		}
	}
}

