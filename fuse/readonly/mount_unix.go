// +build linux darwin freebsd netbsd openbsd
// +build !nofuse

package readonly

import (
	core "github.com/saveio/max/core"
	mount "github.com/saveio/max/fuse/mount"
)

// Mount mounts ONT-IPFS at a given location, and returns a mount.Mount instance.
func Mount(ipfs *core.IpfsNode, mountpoint string) (mount.Mount, error) {
	cfg, err := ipfs.Repo.Config()
	if err != nil {
		return nil, err
	}
	allow_other := cfg.Mounts.FuseAllowOther
	fsys := NewFileSystem(ipfs)
	return mount.NewMount(ipfs.Process(), fsys, mountpoint, allow_other)
}
