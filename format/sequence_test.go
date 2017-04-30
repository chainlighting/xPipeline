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
	"testing"
)

func TestSequence(t *testing.T) {
	expect := shared.NewExpect(t)

	config := core.NewPluginConfig("")
	plugin, err := core.NewPluginWithType("format.Sequence", config)
	expect.NoError(err)

	formatter, casted := plugin.(*Sequence)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("test"), 10)
	result, _ := formatter.Format(msg)

	expect.Equal("10:test", string(result))
}
