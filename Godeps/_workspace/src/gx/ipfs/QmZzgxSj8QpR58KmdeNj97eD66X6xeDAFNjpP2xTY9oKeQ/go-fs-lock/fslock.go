package fslock

import (
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"syscall"

	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmPdKqUcHGFdeSpvjVoaTRPPstGif9GBZb5Q56RVw9o69A/go-ipfs-util"
	logging "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmRREK2CAZ5Re2Bd9zZFG6FeYDppUWt5cMgsoUEp3ktgSr/go-log"
	lock "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmYDxsntDcWC7CTisDmeorEvDVjkRAjjY9Vr9DJ6HSkXMS/go4-lock"
)

// log is the fsrepo logger
var log = logging.Logger("lock")

func errPerm(path string) error {
	return fmt.Errorf("failed to take lock at %s: permission denied", path)
}

// Lock creates the lock.
func Lock(confdir, lockFile string) (io.Closer, error) {
	return lock.Lock(path.Join(confdir, lockFile))
}

// Locked checks if there is a lock already set.
func Locked(confdir, lockFile string) (bool, error) {
	log.Debugf("Checking lock")
	if !util.FileExists(path.Join(confdir, lockFile)) {
		log.Debugf("File doesn't exist: %s", path.Join(confdir, lockFile))
		return false, nil
	}

	lk, err := Lock(confdir, lockFile)
	if err != nil {
		// EAGAIN == someone else has the lock
		if err == syscall.EAGAIN {
			log.Debugf("Someone else has the lock: %s", path.Join(confdir, lockFile))
			return true, nil
		}
		if strings.Contains(err.Error(), "resource temporarily unavailable") {
			log.Debugf("Can't lock file: %s.\n reason: %s", path.Join(confdir, lockFile), err.Error())
			return true, nil
		}

		// we hold the lock ourselves
		if strings.Contains(err.Error(), "already locked") {
			log.Debugf("Lock is already held by us: %s", path.Join(confdir, lockFile))
			return true, nil
		}

		// lock fails on permissions error
		if os.IsPermission(err) {
			log.Debugf("Lock fails on permissions error")
			return false, errPerm(confdir)
		}
		if isLockCreatePermFail(err) {
			log.Debugf("Lock fails on permissions error")
			return false, errPerm(confdir)
		}

		// otherwise, we cant guarantee anything, error out
		return false, err
	}

	log.Debugf("No one has a lock")
	lk.Close()
	return false, nil
}

func isLockCreatePermFail(err error) bool {
	s := err.Error()
	return strings.Contains(s, "Lock Create of") && strings.Contains(s, "permission denied")
}
