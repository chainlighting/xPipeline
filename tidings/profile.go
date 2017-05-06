package tidings

import (
	"github.com/chainlighting/xPipeline/core"
	"github.com/chainlighting/xPipeline/core/log"
	"github.com/chainlighting/xPipeline/shared"
)

type Profile struct {
	base       core.Tidings
	sliceSize  int
	msgMetric  string
}

func init() {
	shared.TypeRegistry.Register(Profile{})
}

func (profile *Profile) Configure(conf core.PluginConfig) error {
	var intVal int
	var strVal string
	var isCvtOk bool
	
	// defautl val
	profile.sliceSize = 512
	profile.msgMetric = "AllMsgCount"

	raw := conf.GetValue("MessageSpecial",nil)
	if nil != raw {
		msgSpecial := raw.(map[interface{}]interface{})

	    intVal, isCvtOk = msgSpecial["SliceSize"].(int)
	    if isCvtOk { profile.sliceSize = intVal}

	    strVal, isCvtOk = msgSpecial["MsgMetricName"].(string)
	    if isCvtOk { profile.msgMetric = strVal}
	}


    shared.Metric.New(profile.msgMetric)

    Log.Debug.Print("configuring tidings.profile: ",profile)

	return nil
}

func (profile *Profile) Identify(data []byte) (msgLen int,startOffset int,padOffset int) {
	return profile.sliceSize,0,0
}

func (profile *Profile) Process(msg []byte,sequence uint64) {
	shared.Metric.Add(profile.msgMetric,1)
}

func (profile *Profile) DumpMetric() {
	val,_ := shared.Metric.Get(profile.msgMetric)
	Log.Note.Print(profile.msgMetric," : ",val)
}