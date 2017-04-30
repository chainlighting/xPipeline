// Copyright 2015-2017 trivago GmbH
//
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

package format

import (
	"github.com/chainlighting/xPipeline/core"
	"github.com/chainlighting/xPipeline/shared"
	"os"
)

// Hostname formatter plugin
// Hostname is a formatter that prefixes a message with the hostname.
// Configuration example
//
//  - "stream.Broadcast":
//    Formatter: "format.Hostname"
//    HostnameFormatter: "format.Envelope"
//    HostnameSeparator: " "
//
// HostnameDataFormatter defines the formatter for the data transferred as
// message. By default this is set to "format.Envelope"
//
// HostnameSeparator sets the separator character placed after the hostname.
// This is set to " " by default.
type Hostname struct {
	base      core.Formatter
	separator string
}

func init() {
	shared.TypeRegistry.Register(Hostname{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *Hostname) Configure(conf core.PluginConfig) error {
	plugin, err := core.NewPluginWithType(conf.GetString("HostnameFormatter", "format.Forward"), conf)
	if err != nil {
		return err
	}

	format.separator = conf.GetString("HostnameSeparator", " ")
	format.base = plugin.(core.Formatter)
	return nil
}

// Format prepends the Hostname of the message to the message.
func (format *Hostname) Format(msg core.Message) ([]byte, core.MessageStreamID) {
	basePayload, stream := format.base.Format(msg)
	baseLength := len(basePayload)

	hostnameStr, err := os.Hostname()
	if err != nil {
		hostnameStr = ""
	} else {
		hostnameStr += format.separator
	}

	payload := make([]byte, len(hostnameStr)+baseLength)
	len := copy(payload, []byte(hostnameStr))
	copy(payload[len:], basePayload)

	return payload, stream
}
