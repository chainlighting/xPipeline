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

package consumer

import (
	"github.com/chainlighting/xPipeline/core"
	"github.com/chainlighting/xPipeline/core/log"
	"github.com/chainlighting/xPipeline/shared"
	"github.com/kr/fs"
	"strings"
	"sync"
	"os"
	"time"
)

const (
	fileBufferGrowSize = 1024
)

type fileState int32

const (
	fileStateDone = fileState(iota)
	fileStateUndone = fileState(iota)
)


//  - "consumer.Files":
//      "Mode": "local"
//      "concurrency": 3
//      "Path": "/var/dir/file"
//      "Remain": true

// msg parser special
const (
	msgSplitAsDefault    = "default"
)

var (
	sizeMetricTags = []string{"0KB","1KB","10KB","100KB","1MB","10MB","100MB","1GB","xGB"}
)

type MsgSplitFunc func(data []byte, dataSize int) (int,int)

type FileProc struct {
	pathfile       string
	fileinfo       os.FileInfo
	state          fileState
	splitFunc      MsgSplitFunc // special msg parse func
}

type LocalFiles struct {
	core.ConsumerBase

	// configuration
	concurrency    int
	currentProcNum int
	concurrencyWg  *sync.WaitGroup	

	path           string
	exceptPath     map[string]bool
	remain         bool
	splitFunc      MsgSplitFunc // special msg parse func

	// statistic
	startTime      time.Time
	filesSizeMetric string
	bytesMetric    string

}

func init() {
	shared.TypeRegistry.Register(LocalFiles{})
}

// Configure initializes this consumer with values from a plugin config.
func (cons *LocalFiles) Configure(conf core.PluginConfig) error {
	err := cons.ConsumerBase.Configure(conf)
	if nil != err {
		return err
	}

	cons.concurrency = conf.GetInt("Concurrency",3)
	if 0 == cons.concurrency {cons.currentProcNum = 1}
	cons.concurrencyWg = &sync.WaitGroup{}

	cons.path = conf.GetString("Path","/var/log/system.log")
	cons.exceptPath = make(map[string]bool)
	ep := conf.GetStringArray("ExceptPath",[]string{})
	for _ , pathName := range ep {
		cons.exceptPath[pathName] = true
	}

	cons.remain = conf.GetBool("Remain",true)

	// set spliter function
	spliter := strings.ToLower(conf.GetString("SpliterAs",msgSplitAsDefault))
	switch spliter {
	case msgSplitAsDefault:
		cons.splitFunc = nil
	default:
		cons.splitFunc = nil
	}

	// statistic setting
	cons.filesSizeMetric = cons.path + ": Size :"
	cons.bytesMetric = cons.path + ": AllBytes :"

    for _ , tagName := range sizeMetricTags {
    	shared.Metric.New(cons.filesSizeMetric + tagName)
    }
    shared.Metric.New(cons.bytesMetric)

	return nil
}

func (cons *LocalFiles) statisticFiles(size int64) {
	ss := size / 1000
	idx := 0
	for ss > 0 {
		idx++
		if idx + 1 >= len(sizeMetricTags) {
			break
		}
		ss = ss / 10
	}
	shared.Metric.Inc(cons.filesSizeMetric + sizeMetricTags[idx])
}

func (cons *LocalFiles) waitConcurrency() {
	if 0 == cons.currentProcNum {
		cons.concurrencyWg.Wait()
		cons.currentProcNum = cons.concurrency
	}
}

func (cons *LocalFiles) goConcurrency(proc *FileProc) {
	cons.currentProcNum--
	cons.concurrencyWg.Add(1)
	go cons.process(proc)
}

func (cons *LocalFiles) process(proc *FileProc) {
	defer cons.concurrencyWg.Done()

	Log.Debug.Print("Processing file: ",proc.fileinfo.Name(),", Size: ",proc.fileinfo.Size())
	cons.statisticFiles(proc.fileinfo.Size())
	shared.Metric.Add(cons.bytesMetric,proc.fileinfo.Size())
}

func (cons *LocalFiles) dumpMetric() {
	allFiles := int64(0)
	Log.Note.Print("Comsumer Start at: ",cons.startTime)
	for _ , tagName := range sizeMetricTags {
		sizeCntVal , _ := shared.Metric.Get(cons.filesSizeMetric + tagName)
		Log.Note.Print(cons.filesSizeMetric + tagName,": ",sizeCntVal)
		allFiles += sizeCntVal
	}
	Log.Note.Print(cons.path,": AllFiles :",allFiles)
	bytesVal , _ := shared.Metric.Get(cons.bytesMetric)
	Log.Note.Print(cons.bytesMetric,": ",bytesVal)

	Log.Note.Print("Consumer cost ",time.Since(cons.startTime))
}

func (cons *LocalFiles) start() {
	defer cons.WorkerDone()
	// start statistic
	cons.startTime = time.Now()
	// enum all files by path
	Log.Debug.Print("Walker start on: ",cons.path)
	walker := fs.Walk(cons.path)
	if nil == walker {
		Log.Error.Print("Failed to walk on: ",cons.path)
		return
	}
	for walker.Step() {
		// concurrency proc wait
		cons.waitConcurrency()
		// process
		if err := walker.Err(); nil != err {
			Log.Error.Print("Walker step failed: ",err)
			continue
		}
        
		if fileinfo := walker.Stat(); fileinfo.IsDir() {
			// diretory proc
			if except , _ := cons.exceptPath[walker.Path()]; except {
				walker.SkipDir()
				continue
			}
		} else {
			// file proc
			if except , _ := cons.exceptPath[walker.Path()]; except {
				continue
			}

			fileproc := &FileProc{
				pathfile: walker.Path(),
				fileinfo: fileinfo,
				state: fileStateUndone,
				splitFunc: cons.splitFunc,
			}
			// concurrency proc start
			cons.goConcurrency(fileproc)
		}
	}
	// concurrency proc wait
	cons.waitConcurrency()
	Log.Debug.Print("Walker complete on: ",cons.path)
	cons.dumpMetric()
}

// Consume listens to stdin.
func (cons *LocalFiles) Consume(workers *sync.WaitGroup) {
	go shared.DontPanic(func() {
		cons.AddMainWorker(workers)
		cons.start()
	})

	cons.ControlLoop()
}

