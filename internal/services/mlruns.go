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
func (d MlrunResourceSrv) PrepareResource(runId string,  resource exports.GObject) (interface{},APIError){

	 access := safeToNumber(resource["access"])
	 if access != 0 {
	 	return nil, exports.NotImplementError("MlrunResourceSrv cannot access by write !")
	 }
	 refer  := safeToString(resource["id"])
	 path,err :=models.CreateLinkWith(runId,refer)
	 if err != nil{
	 	return nil,err
	}
	return map[string]interface{}{
		"path":path,
	},nil
}

// should never error
func (d MlrunResourceSrv) CompleteResource(runId string,resource exports.GObject,commitOrCancel bool) APIError {
	refer  := safeToString(resource["id"])
	return models.DeleteLinkWith(runId,refer)
}

type MlrunOutputSrv struct{

}
//cannot error
func (d MlrunOutputSrv) PrepareResource(runId string, resource exports.GObject) (interface{},APIError){

	refer    := safeToString(resource["id"])
	if refer != "*" {
		return nil,exports.RaiseAPIError(exports.AILAB_LOGIC_ERROR,"MlrunOutputSrv")
	}
	path,err := models.EnsureLabRunStgPath(runId)
	if err != nil {
		return nil,err
	}
	return map[string]interface{}{
		"path":path,
	},nil
}

// should never error
func (d MlrunOutputSrv) CompleteResource(runId string,resource exports.GObject,commitOrCancel bool) APIError {

    return nil
}

func addResource(resource exports.GObject,name string) exports.GObject{
	if v,ok := resource[name];!ok{
		rsc := make(exports.GObject)
		resource[name]=rsc
		return rsc
	}else{
		rsc,_ := v.(exports.GObject)
		if rsc == nil{//should never error
           rsc = make(exports.GObject)
			resource[name]=rsc
		}
		return rsc
	}
}

func ReqCreateRun(labId uint64,parent string,req*exports.CreateJobRequest,enableRepace bool,syncPrepare bool) (interface{},APIError) {

	if len(req.Cmd) == 0 {
		return nil,exports.ParameterError("invalid run cmd !!!")
	}

	//@mark: validate JobRequest fields
	if req.Resource == nil{
		req.Resource = make(exports.GObject)
	}
	//@mark: check add `output` resource type
	if len(req.OutputPath) > 0 {
        rsc := addResource(req.Resource,exports.AILAB_OUTPUT_NAME)
        rsc["id"]     = "*"
        rsc["access"] = 1
	}
	//@mark: check cmds reference valid
	for _,v := range(req.Cmd){
		name,_,_ := fetchCmdResource(v)
		if len(name) == 0 {// not refered resource
			continue
		}
		rsc := addResource(req.Resource,name)
		//validate id
		id  := safeToString(rsc["id"])
		if len(id) == 0{
			if name[0] == '#' { // refer to parent runs
				var err APIError
				rsc["id"],rsc["path"],err = models.RefParentResource(parent,name[1:])
				if err != nil {
					return nil,err
				}
			}else if(name == exports.AILAB_OUTPUT_NAME){//auto allocate output from cmds
				rsc["id"]="*"
				rsc["access"]=1
				req.OutputPath="*"
			}else{
				return nil,exports.ParameterError("invalid cmd refered resource name:" + name)
			}
		}
	}
	if len(req.Resource) == 0 {
		req.Resource=nil
	}
	run,err := models.CreateLabRun(labId,parent,req,enableRepace,syncPrepare)
	if err != nil {
		return run,err
	}
	newRun := run.(*models.Run)
	if newRun.DeletedAt == 0 {
		return run,nil
	}
	err = PrepareResources(newRun,req.Resource,true)
	if err != nil{
		run = nil
	}
	return run,err
}




