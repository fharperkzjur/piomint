
package models

import (
	"database/sql"
	"fmt"
	"github.com/apulis/bmod/ai-lab-backend/internal/configs"
	"github.com/apulis/bmod/ai-lab-backend/internal/utils"
	"github.com/apulis/bmod/ai-lab-backend/pkg/exports"
	"os"
	"path/filepath"
	"strings"
)

func allocateLabStg(lab* Lab) (APIError){

      lab.Location = fmt.Sprintf("%s/%s/%s/%d",exports.AILAB_STORAGE_ROOT,
      	  lab.Namespace,lab.App,lab.ID)

      return nil
}

func allocateLabRunStg(run *Run,mlrun * BasicMLRunContext) (APIError) {

      if run.Output == "*" {//auto generate if not present
      	 run.Output = fmt.Sprintf("%s/%s",mlrun.Location,run.RunId)
	  }
	  return nil
}

func deleteStg(output string) (APIError){
	 if len(output) == 0{
	 	return nil
	 }
	 rpath := mapPVCPath(output)
	 if len(rpath) == 0 {
		 return exports.RaiseAPIError(exports.AILAB_PATH_NOT_FOUND,"pvc path cannot found !")
	 }
	 if err:=os.RemoveAll(rpath);err != nil{
	 	return exports.RaiseAPIError(exports.AILAB_OS_REMOVE_FILE,"delete path error:" + err.Error())
	 }
	 return nil
}

func  mapPVCPath(path string)string{
	 if !strings.HasPrefix(path,"pvc://") {
	 	return ""
	 }
	 index := strings.IndexByte(path[6:],'/')
	 if index < 0 {
	 	return ""
	 }
	 subpath := path[ 6 + index:]
	 pvc_name:= path[6:index+6]
	 rpath := configs.GetAppConfig().Mounts[pvc_name]
	 if len(rpath) == 0 {
	 	return ""
	 }
	 return filepath.Join(rpath,subpath)
}

func EnsureLabRunStgPath(runId string) (path string,err APIError){
	 null_path := sql.NullString{}
	 err = checkDBQueryError(db.Table("runs").Select("output").Where("run_id=?",runId).Row().Scan(&null_path))
	 if err != nil{
	 	return
	 }
	 path = null_path.String
	 if len(path) == 0 {
	 	err = exports.NotFoundError()
	 	return
	 }
	 rpath := mapPVCPath(path)
	 if len(rpath) == 0 {
	 	err = exports.RaiseAPIError(exports.AILAB_PATH_NOT_FOUND,"pvc path cannot found !")
	 	return
	 }
	 mask := utils.Umask(0)  // 改为 0000 八进制
	 defer utils.Umask(mask) // 改为原来的 umask
	 err1 := os.MkdirAll(rpath,os.ModeDir|os.ModePerm)
	 if err1 != nil{
		err= exports.RaiseAPIError(exports.AILAB_OS_CREATE_FILE,"create path error:" + err.Error())
	}
	return
}

