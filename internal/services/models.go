
package services

import (
	"github.com/apulis/bmod/ai-lab-backend/internal/configs"
	"github.com/apulis/bmod/ai-lab-backend/pkg/exports"
)

type ModelResourceSrv struct{

}

const MODEL_MGR_MODULE_ID = 2400

const (
	MODEL_ERROR_CODE_BEGIN = MODEL_MGR_MODULE_ID*100000 + iota
	MODEL_REF_NOT_EXISTS   = MODEL_MGR_MODULE_ID*100000 + 40001          // ref不存在,  ref 和 unref返回
	MODEL_COMMIT_FAIL      = MODEL_MGR_MODULE_ID*100000 + 20002         // commit  新模型失败 ; rollback不会失败
)

type ModelRefInfo struct{
	Context      string         `json:"context,omitempty"`
	ModelID      interface{}    `json:"modelId,omitempty"`
	ModelVersion interface{}    `json:"modelVersionId,omitempty"`
	Scope        string         `json:"scope,omitempty"`
	ModelName    string         `json:"modelName,omitempty"`
}

func checkDebugNoRefPath(resource exports.GObject) interface{} {
	 if configs.GetAppConfig().Debug && safeToString(resource["path"]) != ""{
	 	return map[string]interface{}{
	 		"norefs":1,
		}
	 }else{
	 	return nil
	 }
}
func checkDebugNoRefs(resource exports.GObject) bool {
	 return configs.GetAppConfig().Debug && safeToNumber(resource["norefs"]) == 1
}

//cannot error
func (d ModelResourceSrv) PrepareResource (runId string,  resource exports.GObject) (interface{},APIError){

	  if safeToNumber(resource["access"]) == 0 {//ref exists model
	  	   if norefs := checkDebugNoRefPath(resource) ; norefs != nil{
	  	   	  return norefs,nil
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
	  }else{//save new model
	  	   req := &ModelRefInfo{
			   Context:      runId,
			   ModelID:      resource["id"],
			   ModelVersion: nil,
		   }
		   req.Scope,_     =resource["scope"].(string)
		   req.ModelName,_ =resource["name"].(string)
		   result := make(map[string]interface{})
		   err := Request(configs.GetAppConfig().Resources.Model + "/prepareModel","POST",nil,req, &result)
		   if err != nil {
			  return nil,err
		   }
		  return  result,nil
	}
}

// should never error
func (d ModelResourceSrv) CompleteResource(runId string,resource exports.GObject,commitOrCancel bool) APIError {


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
			if err != nil && err.Errno() == MODEL_COMMIT_FAIL {
				err = exports.RaiseAPIError(exports.AILAB_CANNOT_COMMIT,"commit saved model error:"+err.Error())
			}
			return err
		}
}
