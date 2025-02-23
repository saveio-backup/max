// Copyright 2017 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package flag

import (
	kingpin "gopkg.in/alecthomas/kingpin.v2"
	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmTWEDbLX2d4NiMgPks6J2crRz47BamBtP16WiFuTL6Ydm/common/promlog"
)

// LevelFlagName is the canonical flag name to configure the allowed log level
// within Prometheus projects.
const LevelFlagName = "log.level"

// LevelFlagHelp is the help description for the log.level flag.
const LevelFlagHelp = "Only log messages with the given severity or above. One of: [debug, info, warn, error]"

// AddFlags adds the flags used by this package to the Kingpin application.
// To use the default Kingpin application, call AddFlags(kingpin.CommandLine)
func AddFlags(a *kingpin.Application, logLevel *promlog.AllowedLevel) {
	a.Flag(LevelFlagName, LevelFlagHelp).
		Default("info").SetValue(logLevel)
}
