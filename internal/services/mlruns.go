package services

import (
	"github.com/apulis/bmod/ai-lab-backend/internal/models"
	"github.com/apulis/bmod/ai-lab-backend/pkg/exports"
)

type APIError = exports.APIError

func GetEndpointUrl(mlrun*models.BasicMLRunContext) (interface{},APIError){
	 return "",exports.NotImplementError("GetEndpointUrl")
}


type MlrunResourceSrv struct{

}
//cannot error
func (d MlrunResourceSrv) RefResource(runId string,  resourceId string ,versionId string) (interface{},APIError){



	return nil,exports.NotImplementError("MlrunResourceSrv")
}

// should never error
func (d MlrunResourceSrv) UnRefResource(runId string,resourceId string, versionId string) APIError {


	return exports.NotImplementError("MlrunResourceSrv")
}

func ReqCreateRun(labId uint64,runId string,req*exports.CreateJobRequest,enableRepace bool,syncPrepare bool) (interface{},APIError) {
	run,err := models.CreateLabRun(labId,runId,req,enableRepace,syncPrepare)
	if err != nil {
		return run,err
	}
	newRun := run.(*models.Run)
	if newRun.DeletedAt == nil {
		return run,nil
	}
	err = PrepareResources(newRun,true)
	if err != nil{
		return nil,err
	}else{
		return run,nil
	}
}

func  SyncJobStatus(runId string,status int,msg string) APIError{
	return exports.NotImplementError("NotifyJobStatusChange")
}


