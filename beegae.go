package beegae

import (
	"os"
	"path/filepath"

	"appengine"
)

var TestAppEngineCtx appengine.Context

// this function is for test package init
func TestBeegaeInit(apppath string, testCtx appengine.Context) {
	AppPath = apppath
	RunMode = "test"
	TestAppEngineCtx = testCtx
	AppConfigPath = filepath.Join(AppPath, "conf", "app.conf")
	err := ParseConfig()
	if err != nil && !os.IsNotExist(err) {
		// for init if doesn't have app.conf will not panic
		Info(err)
	}
	os.Chdir(AppPath)
	initBeforeHttpRun()
}
