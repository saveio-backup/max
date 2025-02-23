package commands

import (
	"strings"
	"testing"

	cmds "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmTjNRVt2fvaRFu93keEC7z5M1GS1iH6qZ9227htQioTUY/go-ipfs-cmds"
)

func checkHelptextRecursive(t *testing.T, name []string, c *cmds.Command) {
	if c.Helptext.Tagline == "" {
		t.Errorf("%s has no tagline!", strings.Join(name, " "))
	}

	if c.Helptext.LongDescription == "" {
		t.Errorf("%s has no long description!", strings.Join(name, " "))
	}

	if c.Helptext.ShortDescription == "" {
		t.Errorf("%s has no short description!", strings.Join(name, " "))
	}

	if c.Helptext.Synopsis == "" {
		t.Errorf("%s has no synopsis!", strings.Join(name, " "))
	}

	for subname, sub := range c.Subcommands {
		checkHelptextRecursive(t, append(name, subname), sub)
	}
}

func TestHelptexts(t *testing.T) {
	t.Skip("sill isn't 100%")
	Root.ProcessHelp()
	checkHelptextRecursive(t, []string{"ipfs"}, Root)
}
