
package services

import (
	"github.com/apulis/bmod/ai-lab-backend/internal/models"
	"github.com/apulis/bmod/ai-lab-backend/pkg/exports"
)

type ResourceUsage interface {
	// cannot be error
	RefResource(ctx string,  resourceId string ,versionId string) (interface{},APIError)
	// should never error
	UnRefResource(ctx string,resourceId string, versionId string) APIError
}

type ResourceMgr struct{
	 resources map[string]ResourceUsage
}

func (d ResourceMgr)AddResource(name string,usage ResourceUsage){
	 if d.resources == nil{
	 	d.resources = make(map[string]ResourceUsage)
	 }
	 d.resources[name]=usage
}

var g_resources_mgr ResourceMgr

func init(){

	 g_resources_mgr.AddResource("model",   ModelResourceSrv{})
	 g_resources_mgr.AddResource("dataset", DatasetResourceSrv{})
	 g_resources_mgr.AddResource("mlrun",   MlrunResourceSrv{})
	 g_resources_mgr.AddResource("code",    CodeResourceSrv{})
	 g_resources_mgr.AddResource("engine",  EngineResourceSrv{})
}

func UseResource(ty string, ctx string,resourceId,versionId string) (interface{},APIError) {
	 if usage,ok := g_resources_mgr.resources[ty];ok {
	 	return usage.RefResource(ctx,resourceId,versionId)
	 }else{//should never happen
	 	 logger.Fatalf("cannot support resource type:%s",ty)
		 return nil,exports.NotImplementError("UseResource:"+ty)
	 }
}
func ReleaseResource(ty string,ctx string,resourceId,versionId string)APIError{
	 if usage,ok := g_resources_mgr.resources[ty];ok {
		 return usage.UnRefResource(ctx,resourceId,versionId)
	 }else{//should never happen
		 logger.Fatalf("cannot support resource type:%s",ty)
		 return  exports.NotImplementError("ReleaseResource:"+ty)
 	 }
}

//@mark: input/output
func BatchUseResource(runId string, resource exports.GObject) APIError {
	 if len(resource) == 0 {
        return nil
	 }
	 for k,v := range(resource) {
	 	rsc_cfg , ok := v.(exports.GObject)
	 	if !ok {
	 		return exports.ParameterError("invalid resource definition")
		}
		ty,_ := rsc_cfg["type"].(string)
		if len(ty) == 0 { ty = k }

		usage := g_resources_mgr.resources[ty]
		if usage == nil{
			return exports.NotImplementError("cannot support resource type:" + ty)
		}
		id ,_ := rsc_cfg["id"].(string)
		if len(id) == 0 {
			return exports.ParameterError("invalid resource id")
		}
		version,_:=rsc_cfg["version"].(string)

		resp,err := usage.RefResource(runId,id,version)
		if err != nil{
			logger.Errorf("RefResource[%s]:%s (%s) error:%s",ty,id,version,err.Error())
			return err
		}
		//should never error
		result := resp.(exports.GObject)
		for rk,rv := range(result){// copy to original resource config
			rsc_cfg[rk] = rv
		}
	 }
	 return nil
}

func BatchReleaseResource(run* models.Run) APIError {
	resource := exports.GObject{}
	if err1 := run.Resource.Fetch(&resource) ;err1 != nil {//should never error
		return nil                                         // exports.ParameterError("ReqCreateRun invalid resource definitions !!!")
	}

	if len(resource) == 0 {
		return nil
	}
	for k,v := range(resource) {
		rsc_cfg , ok := v.(exports.GObject)
		if !ok {
			continue
		}
		ty,_ := rsc_cfg["type"].(string)
		if len(ty) == 0 { ty = k }

		usage := g_resources_mgr.resources[ty]
		if usage == nil{
			logger.Errorf("cannot support resource type:" + ty)
			continue
		}
		id ,_ := rsc_cfg["id"].(string)
		if len(id) == 0 {
			logger.Errorf("invalid resource id")
			continue
		}
		version,_:=rsc_cfg["version"].(string)

		err := usage.UnRefResource(run.RunId,id,version)
		if err != nil{
			logger.Errorf("UnRefResource[%s]:%s (%s) error:%s",ty,id,version,err.Error())
			return err
		}
	}
	return nil
}

func PrepareResources(run * models.Run,isRollback bool) (err APIError){
	resource := exports.GObject{}
	if err1 := run.Resource.Fetch(&resource) ;err1 != nil {//should never error
		err = exports.ParameterError("ReqCreateRun invalid resource definitions !!!")
	}
	if err == nil {
		err = BatchUseResource(run.RunId,resource)
	}
	if err == nil{
		run.Resource.Save(resource)
		err = models.PrepareRunSuccess(run.RunId,run.Resource,isRollback)
	}
	if err == nil {//success return
		return nil
	}else if isRollback {//return original error
		models.PrepareRunFailed(run.RunId,isRollback)
		return err
	}else{
		return models.PrepareRunFailed(run.RunId,isRollback)
	}
}
