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
"github.com/apulis/bmod/ai-lab-backend/internal/models"
"github.com/apulis/bmod/ai-lab-backend/pkg/exports"
)

type APWorkshopResourceSrv struct{

}

const APWORKSHOP_MGR_MODULE_ID = 2400

const AISTUDIO_AUTH_FAILED = 500030013

const (
	APWORKSHOP_ERROR_CODE_BEGIN    = APWORKSHOP_MGR_MODULE_ID*100000 + iota
    APWORKSHOP_MODEL_NOT_EXISTS    = APWORKSHOP_MGR_MODULE_ID*100000 + 20101
	APWORKSHOP_REF_NOT_EXISTS      = APWORKSHOP_MGR_MODULE_ID*100000 + 20401         // ref不存在,  ref 和 unref返回
	APWORKSHOP_COMMIT_FAIL         = APWORKSHOP_MGR_MODULE_ID*100000 + 20301         // commit  新模型失败 ; rollback不会失败
	APWORKSHOP_ROLLBACK_NOT_EXISTS = APWORKSHOP_MGR_MODULE_ID*100000 + 20302         // rollback 模型失败 ; rollback不会失败
)

type APWorkshopRefInfo struct{
	Context      string         `json:"context,omitempty"`
	ModelID      int64    `json:"modelId,omitempty"`
	ModelVersion int64    `json:"modelVersionId,omitempty"`
}
type APWorkshopPrepareInfo struct{
	UserName     string         `json:"userName"`
	Context      string         `json:"context,omitempty"`
	Scope        string         `json:"scope"`
	ModelID      int64    `json:"modelId,omitempty"`
	ModelName    string         `json:"modelName"`
	ModelDescription string     `json:"modelDescription,omitempty"`
	IsTmp        bool           `json:"isTmp,omitempty"`
}

//cannot error
func (d APWorkshopResourceSrv) PrepareResource (runId string, token string, resource exports.GObject) (interface{},APIError){

	if safeToNumber(resource["access"]) == 0 {//ref exists model
		if checkDebugNoRefs(resource) {
			return nil,nil
		}
		req :=  &APWorkshopRefInfo{
			Context:      runId,
			ModelID:      safeToNumber(resource["id"]),
			ModelVersion: safeToNumber(resource["version"]),
		}
		result := make(map[string]interface{})
		err := Request(configs.GetAppConfig().Resources.ApWorkshop + "/studioRef","POST",nil,req, &result)
		if err != nil {
			return nil,err
		}
		return result,nil
	}else{//register new model
		req := &APWorkshopPrepareInfo{
			Context:      runId,
			ModelID:      safeToNumber(resource["id"]),
		}
		req.Scope,_     =resource["scope"].(string)
		req.ModelName,_ =resource["name"].(string)
		if len(req.ModelName) == 0 ||  len(req.Scope) == 0{
			scope,name,err := models.GetLabRunDefaultSaveModelName(runId)
			if err != nil {
				return nil,err
			}
			if len(req.Scope) == 0 {
				req.Scope=scope
			}
			if len(req.ModelName) == 0{
				req.ModelName=name
			}
		}

		result := make(map[string]interface{})
		err := Request(configs.GetAppConfig().Resources.ApWorkshop + "/studioPrepareModel","POST",map[string]string{
			"Authorization":"Bearer " + token,
		},req, &result)
		if err != nil {
			return nil,err
		}
		return  result,nil
	}
}

// should never error
func (d APWorkshopResourceSrv) CompleteResource(runId string,token string,resource exports.GObject,commitOrCancel bool) APIError {


	if safeToNumber(resource["access"]) == 0 {//unref model,should never error
		if checkDebugNoRefs(resource) {
			return nil
		}
		req :=  &APWorkshopRefInfo{
			Context:      runId,
			ModelID:      safeToNumber(resource["id"]),
			ModelVersion: safeToNumber(resource["version"]),
		}
		err := Request(configs.GetAppConfig().Resources.ApWorkshop + "/studioUnref?internal=1","POST",nil,req, nil)
		if err != nil && (err.Errno() == APWORKSHOP_REF_NOT_EXISTS || err.Errno() == APWORKSHOP_MODEL_NOT_EXISTS){// unref not exists supress not found error
			err = nil
		}
		return err
	}else{//compete new model
		status := "commit"
		query  := ""
		if commitOrCancel == false {
			status = "rollback"
			query  = "?internal=1"
		}
		err := Request(configs.GetAppConfig().Resources.ApWorkshop + "/studioCommitModel" + query,"PUT",map[string]string{
			"Authorization":"Bearer " + token,
		},
			map[string]interface{}{
				"context" : runId,
				"status":   status,
			}, nil)
		if err != nil {
			if commitOrCancel == true && (err.Errno() == APWORKSHOP_COMMIT_FAIL || err.Errno() == AISTUDIO_AUTH_FAILED){
				err = exports.RaiseAPIError(exports.AILAB_CANNOT_COMMIT,"cannot commit register model :"+err.Error())
			}else if commitOrCancel == false && (err.Errno() == APWORKSHOP_ROLLBACK_NOT_EXISTS ||
				 err.Errno() == APWORKSHOP_REF_NOT_EXISTS || err.Errno() == APWORKSHOP_MODEL_NOT_EXISTS ){
				logger.Warnf("rollback runId: %s with lock:%v not exists !",runId,resource["lock"])
				err = nil
			}
		}
		return err
	}
}
