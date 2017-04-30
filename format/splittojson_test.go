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
	"encoding/json"
	"github.com/chainlighting/xPipeline/core"
	"github.com/chainlighting/xPipeline/shared"
	"testing"
)

func TestSplitToJSON(t *testing.T) {
	expect := shared.NewExpect(t)

	config := core.NewPluginConfig("")
	config.Override("SplitToJSONToken", ",")
	config.Override("SplitToJSONKeys", []string{"first", "second", "third"})

	plugin, err := core.NewPluginWithType("format.SplitToJSON", config)
	expect.NoError(err)

	formatter, casted := plugin.(*SplitToJSON)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("test1,test2,{\"object\": true}"), 10)
	result, _ := formatter.Format(msg)

	jsonData := shared.NewMarshalMap()
	err = json.Unmarshal(result, &jsonData)
	expect.NoError(err)

	expect.MapEqual(jsonData, "first", "test1")
	expect.MapEqual(jsonData, "second", "test2")
	obj, err := jsonData.MarshalMap("third")
	expect.NoError(err)
	expect.MapEqual(obj, "object", true)
}

func TestSplitToJSONTooFew(t *testing.T) {
	expect := shared.NewExpect(t)

	config := core.NewPluginConfig("")
	config.Override("SplitToJSONToken", ",")
	config.Override("SplitToJSONKeys", []string{"first", "second"})

	plugin, err := core.NewPluginWithType("format.SplitToJSON", config)
	expect.NoError(err)

	formatter, casted := plugin.(*SplitToJSON)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("test1,test2,test3"), 10)
	result, _ := formatter.Format(msg)

	jsonData := shared.NewMarshalMap()
	err = json.Unmarshal(result, &jsonData)
	expect.NoError(err)

	expect.MapEqual(jsonData, "first", "test1")
	expect.MapEqual(jsonData, "second", "test2")
	expect.MapNotSet(jsonData, "third")
}

func TestSplitToJSONTooMany(t *testing.T) {
	expect := shared.NewExpect(t)

	config := core.NewPluginConfig("")
	config.Override("SplitToJSONToken", ",")
	config.Override("SplitToJSONKeys", []string{"first", "second", "third", "fourth"})

	plugin, err := core.NewPluginWithType("format.SplitToJSON", config)
	expect.NoError(err)

	formatter, casted := plugin.(*SplitToJSON)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("test1,test2,test3"), 10)
	result, _ := formatter.Format(msg)

	jsonData := shared.NewMarshalMap()
	err = json.Unmarshal(result, &jsonData)
	expect.NoError(err)

	expect.MapEqual(jsonData, "first", "test1")
	expect.MapEqual(jsonData, "second", "test2")
	expect.MapEqual(jsonData, "third", "test3")
}
