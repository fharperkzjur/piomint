
package services

import (
	"github.com/apulis/bmod/ai-lab-backend/internal/configs"
	"github.com/apulis/bmod/ai-lab-backend/pkg/exports"
)

type ModelResourceSrv struct{

}

type ModelRefInfo struct{
	Context      string         `json:"ctx,omitempty"`
	ModelID      interface{}    `json:"modelId,omitempty"`
	ModelVersion interface{}    `json:"modelVersionId,omitempty"`
	Scope        string         `json:"scope,omitempty"`
	ModelName    string         `json:"modelName,omitempty"`
}

//cannot error
func (d ModelResourceSrv) PrepareResource (runId string,  resource exports.GObject) (interface{},APIError){

	  if resource["access"] == 0 {//ref exists model
		   req :=  &ModelRefInfo{
			   Context:      runId,
			   ModelID:      resource["id"],
			   ModelVersion: resource["version"],
		   }
		   path := ""
		   err := Request(configs.GetAppConfig().Resources.Model + "/refs","POST",nil,req, &path)
		   if err != nil {
		   	  return nil,err
		   }
		   return map[string]interface{}{
		   	   "path":path,
		   } ,nil
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


		if resource["access"] == 0 {//unref model,should never error
			req :=  &ModelRefInfo{
				Context:      runId,
				ModelID:      resource["id"],
				ModelVersion: resource["version"],
			}
			return Request(configs.GetAppConfig().Resources.Model + "/unrefs","POST",nil,req, nil)
		}else{//compete new model
			return Request(configs.GetAppConfig().Resources.Model + "/prepareModel","POST",nil,
				  map[string]interface{}{
				      "lock" : resource["lock"],
				  }, nil)
		}
}
