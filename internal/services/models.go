
package services

import (
	"github.com/apulis/bmod/ai-lab-backend/internal/configs"
	"github.com/apulis/bmod/ai-lab-backend/internal/models"
	"github.com/apulis/bmod/ai-lab-backend/pkg/exports"
)

type ModelResourceSrv struct{

}

const MODEL_MGR_MODULE_ID = 2400

const (
	MODEL_ERROR_CODE_BEGIN    = MODEL_MGR_MODULE_ID*100000 + iota
	MODEL_REF_NOT_EXISTS      = MODEL_MGR_MODULE_ID*100000 + 40001          // ref不存在,  ref 和 unref返回
	MODEL_COMMIT_FAIL         = MODEL_MGR_MODULE_ID*100000 + 20002         // commit  新模型失败 ; rollback不会失败
	MODEL_ROLLBACK_NOT_EXISTS = MODEL_MGR_MODULE_ID*100000 + 20003         // rollback 模型失败 ; rollback不会失败
)

type ModelRefInfo struct{
	Context      string         `json:"context,omitempty"`
	ModelID      interface{}    `json:"modelId,omitempty"`
	ModelVersion interface{}    `json:"modelVersionId,omitempty"`
	Scope        string         `json:"scope,omitempty"`
	ModelName    string         `json:"modelName,omitempty"`
}

func checkDebugNoRefs(resource exports.GObject) bool {
	 return safeToNumber(resource[exports.AILAB_RESOURCE_NO_REFS]) == 1
}

//cannot error
func (d ModelResourceSrv) PrepareResource (runId string, token string, resource exports.GObject) (interface{},APIError){

	  if safeToNumber(resource["access"]) == 0 {//ref exists model
	  	   if checkDebugNoRefs(resource) {
	  	   	  return nil,nil
		   }
		   req :=  &ModelRefInfo{
			   Context:      runId,
			   ModelID:      resource["id"],
			   ModelVersion: resource["version"],
		   }
		   result := make(map[string]interface{})
		   err := Request(configs.GetAppConfig().Resources.Model + "/ref","POST",nil,req, &result)
		   if err != nil {
		   	  return nil,err
		   }
		   return result,nil
	  }else{//register new model
	  	   req := &ModelRefInfo{
			   Context:      runId,
			   ModelID:      resource["id"],
			   ModelVersion: nil,
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
		   err := Request(configs.GetAppConfig().Resources.Model + "/prepareModel","POST",nil,req, &result)
		   if err != nil {
			  return nil,err
		   }
		  return  result,nil
	}
}

// should never error
func (d ModelResourceSrv) CompleteResource(runId string,token string,resource exports.GObject,commitOrCancel bool) APIError {


		if safeToNumber(resource["access"]) == 0 {//unref model,should never error
			if checkDebugNoRefs(resource) {
				return nil
			}
			req :=  &ModelRefInfo{
				Context:      runId,
				ModelID:      resource["id"],
				ModelVersion: resource["version"],
			}
			err := Request(configs.GetAppConfig().Resources.Model + "/unref","POST",nil,req, nil)
			if err != nil && err.Errno() == MODEL_REF_NOT_EXISTS {// unref not exists supress not found error
                 err = nil
			}
			return err
		}else{//compete new model
			status := "commit"
			if commitOrCancel == false {
				status = "rollback"
			}
			err := Request(configs.GetAppConfig().Resources.Model + "/commitModel","PUT",nil,
				  map[string]interface{}{
				      "lock" : resource["lock"],
				      "status":status,
				  }, nil)
			if err != nil {
				if commitOrCancel == true && err.Errno() == MODEL_COMMIT_FAIL{
					err = exports.RaiseAPIError(exports.AILAB_CANNOT_COMMIT,"cannot commit register model :"+err.Error())
				}else if commitOrCancel == false && (err.Errno() == MODEL_ROLLBACK_NOT_EXISTS ||
					 err.Errno() == MODEL_COMMIT_FAIL){
					logger.Warnf("rollback runId: %s with lock:%v not exists !",runId,resource["lock"])
                    err = nil
				}
			}
			return err
		}
}
