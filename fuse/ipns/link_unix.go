// +build !nofuse

package ipns

import (
	"context"
	"os"

	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmaFNtBAXX4nVMQWbUqNysXyhevUj1k4B1y5uS45LC7Vw9/fuse"
	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmaFNtBAXX4nVMQWbUqNysXyhevUj1k4B1y5uS45LC7Vw9/fuse/fs"
)

type Link struct {
	Target string
}

func (l *Link) Attr(ctx context.Context, a *fuse.Attr) error {
	log.Debug("Link attr.")
	a.Mode = os.ModeSymlink | 0555
	return nil
}

func (l *Link) Readlink(ctx context.Context, req *fuse.ReadlinkRequest) (string, error) {
	log.Debugf("ReadLink: %s", l.Target)
	return l.Target, nil
}

var _ fs.NodeReadlinker = (*Link)(nil)
