package tidings
import (
	"github.com/chainlighting/xPipeline/core"
	"github.com/chainlighting/xPipeline/core/log"
	"github.com/chainlighting/xPipeline/shared"
)

type Csv struct {
	base       core.Tidings
	header     bool
	spliter    string
}

func init() {
	shared.TypeRegistry.Register(Csv{})
}

func (csv *Csv) Configure(conf core.PluginConfig) error {
	var boolVal bool
	var strVal string
	var isCvtOk bool
	
	// defautl val
	csv.header = false
	csv.spliter = ","

	raw := conf.GetValue("MessageSpecial",nil)
	if nil != raw {
		msgSpecial := raw.(map[interface{}]interface{})

	    boolVal, isCvtOk = msgSpecial["Header"].(bool)
	    if isCvtOk { csv.header = boolVal}

	    strVal, isCvtOk = msgSpecial["Spliter"].(string)
	    if isCvtOk { csv.spliter = strVal}
	}

    Log.Debug.Print("configuring tidings.Csv: ",csv)

	return nil
}

func (csv *Csv) Identify(data []byte) (msgLen int,startOffset int,padOffset int) {
	return 256,0,0
}

func (csv *Csv) Process(msg []byte,sequence uint64) {

}

func (csv *Csv) DumpMetric() {
	
}