// pollEndpoint is a helper utility that waits for a http endpoint to be reachable and return with http.StatusOK
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"time"

	manet "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmRK2LxanhK2gZq6k6R7vk5ZoYZk8ULSSTB7FzDsMUX6CB/go-multiaddr-net"
	logging "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmRb5jh8z2E8hMGN2tkvs1yHynUanqnZ3UeKwgN1i9P1F8/go-log"
	ma "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmWWQ2Txc2c6tqjsBpzg5Ar652cHPGNsQQp2SejkNmkUMb/go-multiaddr"
)

var (
	host     = flag.String("host", "/ip4/127.0.0.1/tcp/5001", "the multiaddr host to dial on")
	endpoint = flag.String("ep", "/version", "which http endpoint path to hit")
	tries    = flag.Int("tries", 10, "how many tries to make before failing")
	timeout  = flag.Duration("tout", time.Second, "how long to wait between attempts")
	verbose  = flag.Bool("v", false, "verbose logging")
)

var log = logging.Logger("pollEndpoint")

func main() {
	flag.Parse()

	// extract address from host flag
	addr, err := ma.NewMultiaddr(*host)
	if err != nil {
		log.Fatal("NewMultiaddr() failed: ", err)
	}
	p := addr.Protocols()
	if len(p) < 2 {
		log.Fatal("need two protocols in host flag (/ip/tcp): ", addr)
	}
	_, host, err := manet.DialArgs(addr)
	if err != nil {
		log.Fatal("manet.DialArgs() failed: ", err)
	}

	if *verbose { // lower log level
		logging.SetDebugLogging()
	}

	// construct url to dial
	var u url.URL
	u.Scheme = "http"
	u.Host = host
	u.Path = *endpoint

	// show what we got
	start := time.Now()
	log.Debugf("starting at %s, tries: %d, timeout: %s, url: %s", start, *tries, *timeout, u)

	for *tries > 0 {

		err := checkOK(http.Get(u.String()))
		if err == nil {
			log.Debugf("ok -  endpoint reachable with %d tries remaining, took %s", *tries, time.Since(start))
			os.Exit(0)
		}
		log.Debug("get failed: ", err)
		time.Sleep(*timeout)
		*tries--
	}

	log.Error("failed.")
	os.Exit(1)
}

func checkOK(resp *http.Response, err error) error {
	if err == nil { // request worked
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			return nil
		}
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Fprintf(os.Stderr, "pollEndpoint: ioutil.ReadAll() Error: %s", err)
		}
		return fmt.Errorf("response not OK. %d %s %q", resp.StatusCode, resp.Status, string(body))
	}
	return err
}
