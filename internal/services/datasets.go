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
package services

import (
	"github.com/apulis/bmod/ai-lab-backend/internal/configs"
	"github.com/apulis/bmod/ai-lab-backend/pkg/exports"
)

type DatasetResourceSrv struct{

}

type DatasetRefInfo struct{
	Context      string         `json:"appCode"`
	DatasetID    int64            `json:"datasetId"`
}

const DATASET_MGR_MODULE_ID = 2200

const (
	DATASET_ERROR_CODE_BEGIN   = DATASET_MGR_MODULE_ID*100000 + iota
	DATASET_NOT_EXISTS         = DATASET_MGR_MODULE_ID*100000 + 30002          // ref不存在,  ref 和 unref返回
	DATASET_CONTEXT_NOT_EXISTS = DATASET_MGR_MODULE_ID*100000 + 30202
)

//cannot error
func (d DatasetResourceSrv) PrepareResource(runId string, token string, resource exports.GObject) (interface{},APIError){

	  if safeToNumber(resource["access"]) != 0 {
	  	  return nil,exports.NotImplementError("DatasetResourceSrv no support for prepare dataset !!!")
	  }
	  if checkDebugNoRefs(resource) {
		return nil,nil
	  }
	  datasetId := safeToNumber(resource["id"])
	  if datasetId == 0 {
	  	return nil, exports.ParameterError("invalid integer dataset id !!!")
	  }
	  req :=  &DatasetRefInfo{
			Context:      runId,
		    DatasetID:    datasetId,
	  }
	  result := make(map[string]interface{})

	  err := Request(configs.GetAppConfig().Resources.Dataset + "/apps/app_code_dataset/bind","POST",nil,req, &result)
	  if err != nil {
		return nil,err
  	  }
	  return map[string]interface{}{
		 "path":result["storagePath"],
	 } ,nil
}

// should never error
func (d DatasetResourceSrv) CompleteResource(runId string, token string,resource exports.GObject,commitOrCancel bool) APIError {

		if safeToNumber(resource["access"]) != 0 {
			return  exports.NotImplementError("DatasetResourceSrv no support for complete dataset !!!")
		}
		if checkDebugNoRefs(resource) {
			return nil
		}
	    datasetId := safeToNumber(resource["id"])
	    if datasetId == 0 {
	    	return nil
	    }
		req :=  &DatasetRefInfo{
			Context:      runId,
			DatasetID:    datasetId,
		}
		err := Request(configs.GetAppConfig().Resources.Dataset + "/apps/app_code_dataset/unbind","POST",nil,req, nil)
		if err != nil && (err.Errno() == DATASET_NOT_EXISTS || err.Errno() ==  DATASET_CONTEXT_NOT_EXISTS){// unref not exists supress not found error
		    err = nil
  	   }
	   return err
}
