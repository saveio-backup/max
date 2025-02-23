// +build darwin,amd64 darwin,arm64 freebsd dragonfly netbsd openbsd

package poll

import (
	"context"
	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmPXvegq26x982cQjSfbTvSzZXn7GiaMwhhVPHkeTEhrPT/sys/unix"
	"time"
)

type Poller struct {
	kqfd  int
	event unix.Kevent_t
}

func New(fd int) (p *Poller, err error) {
	p = &Poller{}

	p.kqfd, err = unix.Kqueue()
	if p.kqfd == -1 || err != nil {
		return nil, err
	}

	p.event = unix.Kevent_t{
		Ident:  uint64(fd),
		Filter: unix.EVFILT_WRITE,
		Flags:  unix.EV_ADD | unix.EV_ENABLE | unix.EV_ONESHOT,
	}
	return p, nil
}

func (p *Poller) Close() error {
	return unix.Close(p.kqfd)
}

func (p *Poller) WaitWriteCtx(ctx context.Context) error {
	deadline, _ := ctx.Deadline()

	// setup timeout
	var timeout *unix.Timespec
	if !deadline.IsZero() {
		d := deadline.Sub(time.Now())
		t := unix.NsecToTimespec(d.Nanoseconds())
		timeout = &t
	}

	// wait on kevent
	events := make([]unix.Kevent_t, 1)
	n, err := unix.Kevent(p.kqfd, []unix.Kevent_t{p.event}, events, timeout)
	if err != nil {
		return err
	}

	if n < 1 {
		return errTimeout
	}
	return nil
}
