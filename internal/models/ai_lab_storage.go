/* ******************************************************************************
* 2019 - present Contributed by Apulis Technology (Shenzhen) Co. LTD
*
* This program and the accompanying materials are made available under the
* terms of the MIT License, which is available at
* https://www.opensource.org/licenses/MIT
*
* See the NOTICE file distributed with this work for additional
* information regarding copyright ownership.
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
* WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
* License for the specific language governing permissions and limitations
* under the License.
*
* SPDX-License-Identifier: MIT
******************************************************************************/
package models

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/apulis/bmod/ai-lab-backend/internal/configs"
	"github.com/apulis/bmod/ai-lab-backend/internal/utils"
	"github.com/apulis/bmod/ai-lab-backend/pkg/exports"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func allocateLabStg(lab* Lab) (APIError){

      lab.Location = fmt.Sprintf("%s/%s/%s/%d",configs.GetAppConfig().Storage,
      	  lab.Namespace,lab.App,lab.ID)

      return nil
}

func allocateLabRunStg(run *Run,mlrun * BasicMLRunContext) (APIError) {

      if run.Output == exports.AILAB_OUTPUT_NAME {//auto generate if not present
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

func ListPathFiles(path ,prefix string)(interface{},APIError){

	if strings.Contains(prefix,"../"){
		return nil,exports.ParameterError("should not have ../ path in prefix !!!")
	}
	path = strings.TrimRight(path,"/")
	prefix=strings.TrimLeft(prefix,"/")
	filePath :=  mapPVCPath(path+"/"+prefix)
	if len(filePath) == 0 {
		return 	nil,exports.RaiseAPIError(exports.AILAB_PATH_NOT_FOUND,"pvc path cannot found !")
	}
	fileInfos,err := ioutil.ReadDir(filePath)
	if err != nil{
		return nil,exports.RaiseAPIError(exports.AILAB_OS_READ_DIR_ERROR,err.Error())
	}
	fileList := make([]exports.FileListItem,0,len(fileInfos))
	for _,item := range fileInfos {
		if item.Name()[0] != '.' {//@todo: supress hidden files & directories all ???
			fileList = append(fileList, exports.FileListItem{
				Name:      item.Name(),
				UpdatedAt: item.ModTime().UnixNano() / 1e6,
				Size:      item.Size(),
				IsDir:     item.IsDir(),
			})
		}
	}
	return fileList,nil
}

func GetRealFilePath(path string,prefix string)(string,APIError){
	path = strings.TrimRight(path,"/")
	prefix=strings.TrimLeft(prefix,"/")
	path =  mapPVCPath(path+"/"+prefix)
	if len(path) == 0 {
		return "",exports.RaiseAPIError(exports.AILAB_FILE_NOT_FOUND,"pvc file path not found !")
	}
	if s,err := os.Stat(path);err != nil {
		return "",exports.RaiseHttpError(http.StatusOK,exports.AILAB_FILE_NOT_FOUND,err.Error())
	}else if(!s.Mode().IsRegular()){
		return "",exports.RaiseAPIError(exports.AILAB_NO_AUTH,"cannot access none regular files!")
	}else{
		return path,nil
	}
}

func WriteJsonFile(dir string,file string,v interface{}) APIError {
	rpath := mapPVCPath(dir)
	if rpath == "" {
		return exports.RaiseAPIError(exports.AILAB_FILE_NOT_FOUND,"pvc file path not found !")
	}
	rpath =  path.Join(rpath,file)
	content,_ := json.Marshal(v)
	if err := ioutil.WriteFile(rpath,content,0444) ;err != nil {
		return exports.RaiseAPIError(exports.AILAB_OS_CREATE_FILE)
	}
	return nil
}
