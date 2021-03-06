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
	"github.com/apulis/bmod/ai-lab-backend/internal/models"
	"github.com/apulis/bmod/ai-lab-backend/pkg/exports"
	"strconv"
	"strings"
)

const (
	resource_release_commit = 0x1
	resource_release_rollback=0x2
	resource_release_readonly=0x4
	resource_release_job_sched=0x8

	resource_release_save = resource_release_commit | resource_release_rollback
)

func needReleaseJobSched(flags int) bool {
	return (flags & resource_release_job_sched) != 0
}
func needReleaseSave(flags int) bool{
    return (flags & (resource_release_commit|resource_release_rollback)) != 0
}
func needReleaseRefs(flags int) bool{
    return (flags & resource_release_readonly) != 0
}

type ResourceUsage interface {
	// cannot be error
	PrepareResource(ctx string, token string, resource exports.GObject) (interface{},APIError)
	// should never error
	CompleteResource(ctx string,token string, resource exports.GObject,commitOrCancel bool ) APIError
}

type ResourceMgr struct{
	 resources map[string]ResourceUsage
}

func (d *ResourceMgr)AddResource(name string,usage ResourceUsage){
	 if d.resources == nil{
	 	d.resources = make(map[string]ResourceUsage)
	 }
	 d.resources[name]=usage
}

var g_resources_mgr ResourceMgr

func init(){

	 g_resources_mgr.AddResource(exports.AILAB_RESOURCE_TYPE_MODEL,   ModelResourceSrv{})
	 g_resources_mgr.AddResource(exports.AILAB_RESOURCE_TYPE_DATASET, DatasetResourceSrv{})
	 g_resources_mgr.AddResource(exports.AILAB_RESOURCE_TYPE_MLRUN,   MlrunResourceSrv{})
	 g_resources_mgr.AddResource(exports.AILAB_RESOURCE_TYPE_CODE,    CodeResourceSrv{})
	 g_resources_mgr.AddResource(exports.AILAB_RESOURCE_TYPE_HARBOR,  HarborResourceSrv{})
	 g_resources_mgr.AddResource(exports.AILAB_RESOURCE_TYPE_OUTPUT, MlrunOutputSrv{} )
	 g_resources_mgr.AddResource(exports.AILAB_RESOURCE_TYPE_STORE,  StoreResourceSrv{})
	 g_resources_mgr.AddResource(exports.AILAB_RESOURCE_TYPE_APWORKSHOP,APWorkshopResourceSrv{})
}

func UseResource(ty string, ctx string,token string, resource exports.GObject) (interface{},APIError) {
	 if usage,ok := g_resources_mgr.resources[ty];ok {
	 	return usage.PrepareResource(ctx,token,resource)
	 }else{//should never happen
		 return nil,exports.NotImplementError("UseResource:"+ty)
	 }
}
func ReleaseResource(ty string,ctx string,token string,resource exports.GObject,cleanFlags int)APIError{
	 if usage,ok := g_resources_mgr.resources[ty];ok {
	 	     if safeToNumber(resource["access"]) == 0 {
                return usage.CompleteResource(ctx,token,resource,false)
			 }else if (resource_release_commit&cleanFlags) != 0 {
	 	 	 	return usage.CompleteResource(ctx,token,resource,true)
			 }else if (resource_release_rollback&cleanFlags)!= 0{
			 	return usage.CompleteResource(ctx,token,resource,false)
			 }else  {//trival
			 	return nil
			 }
	 }else{//should never happen
		 return  exports.NotImplementError("ReleaseResource:"+ty)
 	 }
}

func fetchCmdResource(value string)(string,string,string){
	if len(value) > 4 && value[0] == '{' && value[1] == '{' {
		 if index := strings.Index(value,"}}");index > 0 {
             name := value[2:index]
             name  = strings.TrimSpace(name)
             names := strings.SplitN(name,"/",2)
             if len(names) == 1 {
             	return names[0],"",value[index+2:]
			 }else if len(names) == 2{
				 return names[0],names[1],value[index+2:]
			 }else{
			 	panic("strings.SplitN 2 logic error!")
			 }
		 }
	}
	return "","",""
}
func safeToString(v interface{})string{
	switch n:=v.(type){
	   case string: return n
	   case  int64: return strconv.FormatInt(n,10)
	   case float64:return strconv.FormatInt(int64(n),10)
	   default:     return ""
	}
}
func safeToNumber(v interface{}) (value int64){
	 switch n:=v.(type){
	  case string: value,_ = strconv.ParseInt(n,0,32)
	  case int64:  value = n
	  case int:     value=int64(n)
	  case  float64:value=int64(n)
	  default:       value=0
	 }
	 return
}

//@mark: input/output
func BatchUseResource(runId string,token string,resource exports.GObject) APIError {

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

		 if ty[0] == '#'{//ref parent does not need to ref&unref
		 	 rsc_cfg["rpath"]=getPVCMappedPath(k,"",safeToString(rsc_cfg["rpath"]))
			 continue
		 }
		 //if id  := safeToString(rsc_cfg["id"]);len(id)==0 {
		//	 return exports.ParameterError("invalid resource id with name:" + k)
		//}
		 resp,err := UseResource(ty,runId,token,rsc_cfg)
		 if err != nil{
			logger.Errorf("RefResource[%s]: error:%s",ty,err.Error())
			return err
		 }
		 //should never error
		 if result,ok := resp.(exports.GObject);ok {
			 for rk,rv := range(result){// copy to original resource config
				 rsc_cfg[rk] = rv
			 }
		 }
		 //fill in rpath to mounts into pods
		 if path,_ := rsc_cfg["path"].(string);checkIsPVCURL(path){
		 	rsc_cfg["rpath"]=getPVCMappedPath(k,"",safeToString(rsc_cfg["rpath"]))
		 }
	 }
	 return nil
}

func BatchReleaseResource(run* models.Run,commitFlags int) APIError {
	if run.Resource == nil || commitFlags == 0{
		return nil
	}
	resource := exports.GObject{}
	if err1 := run.Resource.Fetch(&resource) ;err1 != nil {//should never error
		return  exports.ParameterError("BatchReleaseResource invalid resource definitions !!!")
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

		if ty[0] == '#'{//ref parent does not need to ref&unref
			continue
		}

		err := ReleaseResource(ty,run.RunId,run.Token, rsc_cfg,commitFlags)
		if err != nil {
			if err.Errno() == exports.AILAB_NOT_IMPLEMENT {
				logger.Errorf("cannot support resource type:" + ty)
				continue
			}else{
				logger.Errorf("UnRefResource[%s] type:%s error:%s",k,ty,err.Error())
				return err
			}
		}
	}
	return nil
}

func PrepareResources(run * models.Run,resource exports.GObject, isRollback bool) (err APIError){
	if resource == nil{
		if err := run.Resource.Fetch(&resource) ;err != nil {//should never error
			return exports.ParameterError("PrepareResources invalid resource definitions !!!")
		}
	}
	if len(resource) == 0 {// nothing need to prepare
		return nil
	}

	if err == nil {
		err = BatchUseResource(run.RunId,run.Token,resource)
	}
	if err == nil{
		run.Resource.Save(resource)
		//@add: save run data as output file if output dir exists
		if output := run.Output;output != "" {
			 run.Output=""
			 run.Token=""
			 err = models.WriteJsonFile(output,exports.AILAB_OUTPUT_RUN_FILE,run)
		}
		if err == nil {
			err = models.PrepareRunSuccess(run.RunId,run.Resource,isRollback)
		}
	}
	if err == nil {//success return
		return nil
	}else if isRollback {//return original error
		models.PrepareRunFailed(run.RunId,err.Error(),isRollback)
		return err
	}else if err.Errno() == exports.AILAB_REMOTE_NETWORK_ERROR{// call from backend events queue, try next timer
		return err
	}else{
		return models.PrepareRunFailed(run.RunId,err.Error(),isRollback)
	}
}
