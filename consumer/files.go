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
	"sync"
	"os"
	"io"
	"time"
	"path/filepath"
)

const (
	fileBufferGrowSize = 4096000
	fileConcurrencyNum = 3
)

type fileState int32

const (
	fileStateDone = fileState(iota)
	fileStateUndone = fileState(iota)
)


//  - "consumer.Files":
//      "concurrency": 3
//      "Path": "/var/dir/file"
//      "Remain": true

var (
	sizeMetricTags = []string{"0KB","1KB","10KB","100KB","1MB","10MB","100MB","1GB","xGB"}
)

type FileProc struct {
	pathfile       string
	fileinfo       os.FileInfo
	state          fileState
}

type Files struct {
	core.ConsumerBase

	// configuration
	readBufferGrowSize int
	concurrency        int
	currentProcNum     int
	concurrencyWg      *sync.WaitGroup	

	path               string
	exceptPath         map[string]bool
	acceptFile         map[string]bool
	remain             bool

	keepRunning        bool

	// statistic
	startTime          time.Time
	filesSizeMetric    string
	bytesMetric        string
	msgMetric          string

}

func init() {
	shared.TypeRegistry.Register(Files{})
}

// Configure initializes this consumer with values from a plugin config.
func (cons *Files) Configure(conf core.PluginConfig) error {
	err := cons.ConsumerBase.Configure(conf)
	if nil != err {
		return err
	}

	cons.concurrency = conf.GetInt("Concurrency",fileConcurrencyNum)
	if 0 == cons.concurrency {cons.currentProcNum = 1}
	cons.concurrencyWg = &sync.WaitGroup{}

	cons.readBufferGrowSize = conf.GetInt("ReadBufferGrowSize",fileBufferGrowSize)

	cons.path = conf.GetString("RootPath","/var/log/system.log")

	cons.exceptPath = make(map[string]bool)
	ep := conf.GetStringArray("ExceptPaths",[]string{})
	for _ , exPathName := range ep {
		cons.exceptPath[exPathName] = true
	}

    cons.acceptFile = make(map[string]bool)
	ap := conf.GetStringArray("AcceptFiles",[]string{})
	for _ , acFileName := range ap {
		cons.acceptFile[acFileName] = true
	}

	cons.remain = conf.GetBool("Remain",true)
	cons.keepRunning = conf.GetBool("KeepRunning",true)

	// statistic setting
	cons.filesSizeMetric = cons.path + ": Size :"
	cons.bytesMetric = cons.path + ": AllBytes :"
	cons.msgMetric = cons.path + ": MessageNum :"

    for _ , tagName := range sizeMetricTags {
    	shared.Metric.New(cons.filesSizeMetric + tagName)
    }
    shared.Metric.New(cons.bytesMetric)
    shared.Metric.New(cons.msgMetric)

	return nil
}

func (cons *Files) statisticFiles(size int64) {
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

func (cons *Files) dumpMetric() {
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

	cons.Gtidings.DumpMetric()

	Log.Note.Print("Consumer cost ",time.Since(cons.startTime))
}

func (cons *Files) getFilepath(path string) string{
	baseFilepath, err := filepath.EvalSymlinks(path)
	if err != nil {
		baseFilepath = path
	}

	baseFilepath, err = filepath.Abs(baseFilepath)
	if err != nil {
		baseFilepath = path
	}

	return baseFilepath
}

func (cons *Files) skipFilepath(path string,isFile bool) bool {
	var match bool
	var defAction bool

    if isFile {
    	defAction = false
    	if len(cons.acceptFile) > 0 {
    	    defAction = true
    	}
	    for ap,_ := range cons.acceptFile {		    
    		match, _ = filepath.Match(ap,filepath.Base(path))
	    	if match {
	    		Log.Debug.Print("Match Accept: ",ap,"<-->",path)
		    	return false
		    }
	    }
	    return defAction
	} else {
		defAction = false
		for ep,_ := range cons.exceptPath {		    
		    match, _ = filepath.Match(ep,path)
		    if match {
		    	Log.Debug.Print("Match Except: ",ep,"<-->",path)
			    return true
		    }
	    }
	    return defAction
	}
	
	return false
}

func (cons *Files) monitorConcurrency() {
	if 0 == cons.currentProcNum {
		cons.concurrencyWg.Wait()
		cons.currentProcNum = cons.concurrency
	}
}

func (cons *Files) waitAllComplete() {
	cons.concurrencyWg.Wait()
	cons.currentProcNum = cons.concurrency
}

func (cons *Files) goConcurrency(proc *FileProc) {
	cons.currentProcNum--
	cons.concurrencyWg.Add(1)
	go cons.process(proc)
}

func (cons *Files) streamRead(proc *FileProc) {
	//buffer := shared.NewBufferedReader(cons.readBufferGrowSize, 0, 0, "",cons.msgSpliter)
	buffer := shared.NewBufferedReader(cons.readBufferGrowSize, 0, 0, "",cons.Gtidings.Identify)
	file, err := os.OpenFile(cons.getFilepath(proc.pathfile), os.O_RDONLY, 0666)
	if nil != err {
		Log.Error.Print("Failed to open file: ",proc.pathfile)
	}

    defer file.Close()

	for {
		//err = buffer.ReadAll(file, cons.msgProc)
		err = buffer.ReadAll(file, cons.Gtidings.Process)
	    if io.EOF == err {
		    Log.Debug.Print("Process success on file: ",proc.pathfile)
		    return
		} else if nil != err {
		    Log.Debug.Print("Process failed on file: ",proc.pathfile)
		    return
	    }
	}
	
}

func (cons *Files) process(proc *FileProc) {
	defer cons.concurrencyWg.Done()
    
    Log.Debug.Print("Processing file: ",proc.pathfile,", Size: ",proc.fileinfo.Size())
	
	cons.streamRead(proc)	
	cons.statisticFiles(proc.fileinfo.Size())
	shared.Metric.Add(cons.bytesMetric,proc.fileinfo.Size())
}

func (cons *Files) start() {
	defer cons.WorkerDone()
	// start statistic
	cons.startTime = time.Now()
	// enum all files by path
	Log.Debug.Print("Walker start on: ",cons.path)
	walker := fs.Walk(cons.getFilepath(cons.path))
	if nil == walker {
		Log.Error.Print("Failed to walk on: ",cons.path)
		return
	}
	for walker.Step() {
		// concurrency proc wait
		cons.monitorConcurrency()
		// process
		if err := walker.Err(); nil != err {
			Log.Error.Print("Walker step failed: ",err)
			continue
		}
        
		if fileinfo := walker.Stat(); fileinfo.IsDir() {
			// diretory proc
			if cons.skipFilepath(walker.Path(),false) {
				walker.SkipDir()
				continue
			}
		} else {
			// file proc
			if cons.skipFilepath(walker.Path(),true) {
				continue
			}

			// concurrency proc start
			fileproc := &FileProc{
				pathfile: walker.Path(),
				fileinfo: fileinfo,
				state: fileStateUndone,
			}
			cons.goConcurrency(fileproc)
		}
	}
	// walk complete wait
	cons.waitAllComplete()
	Log.Debug.Print("Walker complete on: ",cons.path)
	cons.dumpMetric()

    // **this method is not graceful,Producer may not be ended
	if !cons.keepRunning {
		proc, _ := os.FindProcess(os.Getpid())
		proc.Signal(os.Interrupt)
	}
}

// Consume root routine.
func (cons *Files) Consume(workers *sync.WaitGroup) {
	go shared.DontPanic(func() {
		cons.AddMainWorker(workers)
		cons.start()
	})

	cons.ControlLoop()
}

