package services

import (
	"encoding/base64"
	"github.com/apulis/bmod/ai-lab-backend/internal/models"
	"github.com/apulis/bmod/ai-lab-backend/internal/utils"
	"github.com/apulis/bmod/ai-lab-backend/pkg/exports"
)


func GetEndpointUrl(mlrun*models.BasicMLRunContext,name string) (interface{},APIError){
	if mlrun.StatusIsRunning() {
		if mlrun.Endpoints == nil {
			return nil,exports.RaiseAPIError(exports.AILAB_LOGIC_ERROR,"no endpoints created for run:"+mlrun.RunId)
		}
		endpoints := models.UserEndpointList{}
		if err:=mlrun.Endpoints.Fetch(&endpoints);err != nil {
			return nil,exports.RaiseAPIError(exports.AILAB_LOGIC_ERROR,"invalid endpoints data formats for run:"+mlrun.RunId)
		}
		if userEndpoint,_ := endpoints.FindEndpoint(name);userEndpoint == nil {
			return nil,exports.NotFoundError()
		}else{
			return userEndpoint.GetAccessEndpoint(mlrun.Namespace).Url,nil
		}
	}else{
		return "",exports.RaiseReqWouldBlock("wait to starting job ...")
	}
}
func GetLabRunEndpoints(labId uint64,runId string,fetchInternal bool) (interface{},APIError){
	run, err := models.QueryRunDetail(runId,false,0,false)
	if err != nil {
		return nil,err
	}
	if run.LabId != labId {
		return nil,exports.RaiseAPIError(exports.AILAB_LOGIC_ERROR,"invalid lab id passed for runs")
	}
	if run.StatusIsNonActive() {
		return nil,exports.RaiseAPIError(exports.AILAB_INVALID_RUN_STATUS,"lab run is non-active yet !")
	}else if fetchInternal || run.StatusIsRunning() {
		if run.Endpoints == nil {
			return nil,exports.RaiseAPIError(exports.AILAB_LOGIC_ERROR,"no endpoints created for run:"+run.RunId)
		}
		endpoints := models.UserEndpointList{}
		if err:=run.Endpoints.Fetch(&endpoints);err != nil {
			return nil,exports.RaiseAPIError(exports.AILAB_LOGIC_ERROR,"invalid endpoints data formats for run:"+run.RunId)
		}
		response := []exports.ServiceEndpoint{}
		for _,v := range(endpoints) {
			svc := v.GetAccessEndpoint(run.Namespace)
			if fetchInternal {
				svc.ServiceName=v.ServiceName
			}
			response=append(response,svc)
		}
		return response,nil
	}else{
		return "",exports.RaiseReqWouldBlock("wait to starting job ...")
	}
}

func CreateLabRunEndpoints(labId uint64,runId string,endpoint*exports.ServiceEndpoint) APIError {
	// update db first
	mlrun,userEndpoint,err := models.CreateUserEndpoint(labId,runId,endpoint)
	if err != nil {
		return err
	}
	if userEndpoint.Status == exports.AILAB_USER_ENDPOINT_STATUS_INIT || userEndpoint.Status == exports.AILAB_USER_ENDPOINT_STATUS_ERROR {
		//retry create endpoint
		err1, status := CreateJobEndpointImpl(mlrun,userEndpoint)
		err = models.CompleteUserEndpoint(labId,runId,userEndpoint.Name,userEndpoint.Status,status)
		if err == nil {
			err = err1
		}
		return err
	}else if userEndpoint.Status == exports.AILAB_USER_ENDPOINT_STATUS_STOP {
		return exports.RaiseAPIError(exports.AILAB_INVALID_ENDPOINT_STATUS,"endpoint still stopping !")
	}else{
		return nil
	}
}

func DeleteLabRunEndpoints(labId uint64,runId string,name string) APIError{
	// update db first
	mlrun,userEndpoint,err := models.DeleteUserEndpoint(labId,runId,name)
	if err != nil {
		return err
	}
	// try delete endpoint
	err1,status := DeleteJobEndPointImpl(mlrun,userEndpoint.ServiceName)
	err =  models.CompleteUserEndpoint(labId,runId,name,exports.AILAB_USER_ENDPOINT_STATUS_STOP,status)
	if err == nil {
		err = err1
	}
	return err
}

func ValidateUserEndpoints( req []exports.ServiceEndpoint) APIError {
	if req == nil {
		return nil
	}
	names  := make(map[string]int,0)
	ports  := make(map[int]int,0)
	for idx,v := range(req) {
		if _,ok := names[v.Name];ok {
			return exports.ParameterError("duplicate endpoints name !!!")
		}else{
			names[v.Name]=1
		}
		if _,ok := ports[v.Port];v.Port > 0 && ok {
			return exports.ParameterError("duplicate endpoints port !!!")
		}else{
			ports[v.Port]=1
		}
		if v.SecretKey == exports.AILAB_SECURE_DEFAULT  {// need generate passwd by AILAB
			req[idx].SecretKey= base64.StdEncoding.EncodeToString(utils.GenerateRandomPasswd(8))
		}
	}
	return nil
}
