
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
func (d DatasetResourceSrv) PrepareResource(runId string,  resource exports.GObject) (interface{},APIError){

	  if safeToNumber(resource["access"]) != 0 {
	  	  return nil,exports.NotImplementError("DatasetResourceSrv no support for prepare dataset !!!")
	  }
	  if norefs := checkDebugNoRefPath(resource) ; norefs != nil{
		return norefs,nil
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
func (d DatasetResourceSrv) CompleteResource(runId string,resource exports.GObject,commitOrCancel bool) APIError {

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
