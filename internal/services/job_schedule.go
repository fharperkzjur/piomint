
package services

import (
	"encoding/json"
	"fmt"
	"github.com/apulis/bmod/ai-lab-backend/internal/configs"
	"github.com/apulis/bmod/ai-lab-backend/internal/models"
	"github.com/apulis/bmod/ai-lab-backend/pkg/exports"
    JOB "github.com/apulis/go-business/pkg/jobscheduler"
	"github.com/apulis/sdk/go-utils/broker"
	"strings"
)

func checkIsPVCURL(path string)bool{
	return strings.HasPrefix(path,"pvc://")
}
func getPVCMountPath(mount_name,rpath string) string{
	if len(rpath) == 0 {//use default mount path
		rpath = exports.AILAB_DEFAULT_MOUNT
	}
	if strings.HasPrefix(mount_name,exports.AILAB_OUTPUT_NAME) {
		mount_name = exports.AILAB_OUTPUT_MOUNT + mount_name[len(exports.AILAB_OUTPUT_NAME):]
	}else if mount_name[0] =='#'{// mount to refer parent resource
        mount_name = exports.AILAB_PIPELINE_REFER_PREFIX + mount_name[1:]
	}
	rpath += "/" + mount_name
	return strings.TrimRight(rpath,"/")
}

func translateJobStatus(status JOB.JobStatus) int{
	switch(status){
	    case JOB.JOB_STATUS_QUEUEING:    return exports.AILAB_RUN_STATUS_QUEUE
	    case JOB.JOB_STATUS_SCHEDULING:  return exports.AILAB_RUN_STATUS_SCHEDULE
	    case JOB.JOB_STATUS_RUNNING:     return exports.AILAB_RUN_STATUS_RUN
	    case JOB.JOB_STATUS_FINISH:      return exports.AILAB_RUN_STATUS_SUCCESS
	    case JOB.JOB_STATUS_ERROR:       return exports.AILAB_RUN_STATUS_ERROR
	    default://should never happen
	         logger.Warnf("invalid status return from job-sched:%v",status)
	         return -1
	}
}

func tryResourceMounts(name,path,rpath string,access int , subname,subpath string,maps map[string]int,
	          mounts []JOB.MountPoint) ([]JOB.MountPoint,string){
	  mount_name := name + "/" + subname
	  pvc_path   := path
	  if len(subpath) > 0 {
	  	pvc_path = subpath
	  }
	  if maps[mount_name] == 0 && checkIsPVCURL(pvc_path) {
		  mounts = append(mounts,JOB.MountPoint{
			  Path:          pvc_path,
			  ContainerPath: getPVCMountPath(mount_name,rpath),
			  ReadOnly:      access == 0,
		  })
		  maps[mount_name]=1//track resource mounted
	  }
	  if(checkIsPVCURL(pvc_path)){
		return mounts,getPVCMountPath(mount_name,rpath)
	  }else if len(subname) > 0 {
		return mounts,pvc_path + "/" + subname
	  }else{
	  	return mounts,pvc_path
	  }
}

//should never error here
func checkResourceMounts(cmds []string,resources exports.GObject) ([]string, []JOB.MountPoint){
	 action := make([]string,len(cmds))
	 mounts := []JOB.MountPoint{}

	 maps   := make(map[string]int)

	 for _,value := range(cmds){
	 	name,subname,extra := fetchCmdResource(value)
	 	if rsc,ok :=resources[name].(exports.GObject);ok && len(name) > 0 {
	 		path,_    := rsc["path"].(string)
	 		rpath,_   := rsc["rpath"].(string)
	 		subpath   := ""
	 		access,_  := rsc["access"].(int)

	 		if len(subname) > 0 {// has subsource
	 			subresource,_ := rsc["subResource"].(exports.GObject)
	 			if subresource != nil{
	 				subpath,_ = subresource[subname].(string)
				}
			}
			mounts,value = tryResourceMounts(name,path,rpath,access,subname,subpath,maps,mounts)
	 		action=append(action,value + extra)
		}else{
			action=append(action,value)
		}
	 }
	 for name,value := range(resources) {
	 	 if rsc,ok := value.(exports.GObject);ok {
	 	 	 if path,_ := rsc["path"].(string);checkIsPVCURL(path){
				 rpath,_    := rsc["rpath"].(string)
				 access,_  := rsc["access"].(int)

				 mounts,_ = tryResourceMounts(name,path,rpath,access,"","",maps,mounts)

			 }
		 }
	 }
	 return action,mounts
}

func SubmitJob(run*models.Run) (int, APIError) {

	 url  := configs.GetAppConfig().Resources.Jobsched

	 job := &JOB.Job{
		 ModId:       exports.AILAB_MODULE_ID,
		 JobId:       run.RunId,
		 Owner:       run.Creator,
		 ResType:     JOB.RESOURCE_TYPE_JOB,
		 ImageName:   run.Image,
		 Namespace:   run.Namespace,
		 MountPoints: make([]JOB.MountPoint,0),
		 ArchType:    run.Arch,
	 }
	 resource := exports.GObject{}
	 run.Cmd.Fetch(&job.Cmd)
	 run.Envs.Fetch(&job.Envs)
 	 run.Resource.Fetch(&resource)
	 run.Quota.Fetch(&job.Quota)
	 ports :=[]JOB.ContainerPort{}
	 run.Endpoints.Fetch(&ports)
	 if len(ports) > 0 {
	 	job.SetContainerPorts(ports)
	 }
	 job.Cmd,job.MountPoints = checkResourceMounts(job.Cmd,resource)
	 //@todo:  add pre-start scripts ???
	 job.PreStartScripts="*"
	 resp := &JOB.CreateJobRsp{}

	 err  := Request(url,"POST",nil,job,resp)
	 if err == nil {// no error occur
	 	   return translateJobStatus(resp.JobState.Status),nil
	 }else if err.Errno() == JOB.ERR_CODE_RESOURCE_EXIST{// create job already exists
        //@todo: will return exists status also ???
	 	status := translateJobStatus(resp.JobState.Status)
	 	if status > 0 && exports.IsRunStatusActive(status) {// still active yet
           return status,nil
		}
        if run.EnableResume(){//try delete job and resume again
         	err = DeleteJob(run.RunId)
         	logger.Warnf("try submit resumable job:%s already exists , delete first ...",run.RunId)
         	if err != nil {
         		return -1,err
			}
			//submit again
			err = Request(url,"POST",nil,job,resp)
			if err == nil {
				return translateJobStatus(resp.JobState.Status),nil
			}else if err.Errno() == JOB.ERR_CODE_INVALID_PARAM{
				return exports.AILAB_RUN_STATUS_FAILED,err
			}else{
				return -1,err
			}
		 }else{//should not change run status here
		 	logger.Warnf("try submit job:%s already exists!",run.RunId)
            return exports.AILAB_RUN_STATUS_FAILED,err
		 }
	 }else if err.Errno() == JOB.ERR_CODE_INVALID_PARAM{
	 	 return exports.AILAB_RUN_STATUS_FAILED,err
	 }else{
	 	 return -1,err
	 }
}

func KillJob(run*models.Run) (int,APIError) {
	 if err := DeleteJob(run.RunId);err == nil{
	 	return exports.AILAB_RUN_STATUS_ABORT,nil
	 }else{
	 	return -1,err
	 }
}

func DeleteJob(runId string) APIError{
	 url  := fmt.Sprintf("%s/%s",configs.GetAppConfig().Resources.Jobsched,runId)
	 err  := Request(url,"DELETE",nil,nil,nil)
	 if err == nil || err.Errno() == JOB.ERR_CODE_RESOURCE_NOT_EXIST {//should never error
		return nil
	 }else{
		return err
	 }
}

func  SyncJobStatus(runId string,statusFrom int, statusTo int,err APIError) APIError{
	 if statusTo < 0 {//cancel this action
	 	return err
	 }
	 msg := ""
	 if err != nil {
	 	msg=err.Error()
	 }
	 return models.ChangeJobStatus(runId,statusFrom,statusTo,msg)
}

func  MonitorJobStatus(event broker.Event) error{
	  jobs := &JOB.JobMsg{}
	  if json.Unmarshal(event.Message().Body,jobs) == nil {
		  statusTo := translateJobStatus(jobs.JobState.Status)
		  if statusTo < 0 {
		  	logger.Warnf("receive unknown job state from mq:%s",event.Message().Body)
		  }else{
		  	return models.ChangeJobStatus(jobs.JobId,-1,statusTo,jobs.JobState.Msg)
		  }

	  }
	  return nil
}

