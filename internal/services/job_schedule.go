
package services

import (
	"encoding/json"
	"fmt"
	"github.com/apulis/bmod/ai-lab-backend/internal/configs"
	"github.com/apulis/bmod/ai-lab-backend/internal/models"
	"github.com/apulis/bmod/ai-lab-backend/internal/utils"
	"github.com/apulis/bmod/ai-lab-backend/pkg/exports"
    JOB "github.com/apulis/go-business/pkg/jobscheduler"
	"github.com/apulis/sdk/go-utils/broker"
	"strings"
)

func checkIsPVCURL(path string)bool{
	return strings.HasPrefix(path,"pvc://")
}
func getPVCMappedPath(name,subname,rpath string) string{
	if strings.HasPrefix(name,exports.AILAB_OUTPUT_NAME) {
		name = exports.AILAB_OUTPUT_MOUNT
	}else if name[0] =='#'{// mount to refer parent resource
		name = exports.AILAB_PIPELINE_REFER_PREFIX + name[1:]
	}
	if len(rpath) == 0 {//use default mount path
		rpath = exports.AILAB_DEFAULT_MOUNT + "/" + name
	}
	if len(subname) == 0 {
		return rpath
	}else{
		return rpath + "/" + subname
	}
}
func getPVCMountedPath(name,rpath string,subname ,subpath string) string{
	if strings.HasPrefix(name,exports.AILAB_OUTPUT_NAME) {
		name = exports.AILAB_OUTPUT_MOUNT
	}else if name[0] =='#'{// mount to refer parent resource
		name = exports.AILAB_PIPELINE_REFER_PREFIX + name[1:]
	}
	if len(rpath) == 0 {//use default mount path
		rpath = exports.AILAB_DEFAULT_MOUNT + "/" + name
	}
	if len(subpath) == 0 {
		return rpath
	}else{
		return rpath + "/" + subname
	}
}

func translateJobStatus(status JOB.JobStatus) int{
	switch(status){
	    case JOB.JOB_STATUS_UNAPPROVE:   return exports.AILAB_RUN_STATUS_QUEUE
	    case JOB.JOB_STATUS_QUEUEING:    return exports.AILAB_RUN_STATUS_QUEUE
	    case JOB.JOB_STATUS_SCHEDULING:  return exports.AILAB_RUN_STATUS_SCHEDULE
	    case JOB.JOB_STATUS_RUNNING:     return exports.AILAB_RUN_STATUS_RUN
	    case JOB.JOB_STATUS_FINISH:      return exports.AILAB_RUN_STATUS_SUCCESS
	    case JOB.JOB_STATUS_ERROR:       return exports.AILAB_RUN_STATUS_ERROR
	    default://should never happen
	         logger.Warnf("invalid status return from job-sched:%v",status)
	         return exports.AILAB_RUN_STATUS_INVALID
	}
}

func tryResourceMounts(name,path,rpath string,access int , subname,subpath string,maps map[string]int,
	          mounts []JOB.MountPoint) ([]JOB.MountPoint,string){
	  mount_name := name
	  pvc_path   := path
	  if len(subpath) > 0 {
	  	pvc_path = subpath
	  	mount_name = name + "/" + subname
	  }
	  if maps[mount_name] == 0 && checkIsPVCURL(pvc_path) {
		  mounts = append(mounts,JOB.MountPoint{
			  Path:          pvc_path,
			  ContainerPath: getPVCMountedPath(name,rpath,subname,subpath),
			  ReadOnly:      access == 0,
		  })
		  maps[mount_name]=1//track resource mounted
	  }
	  if(checkIsPVCURL(pvc_path)){
		return mounts,getPVCMappedPath(name,subname,rpath)
	  }else if len(subname) > 0 {
	  	if len(subpath) >0 {
            return mounts,subpath
		}else{
			return mounts,pvc_path + "/" + subname
		}
	  }else{
	  	return mounts,pvc_path
	  }
}

//should never error here
func CheckResourceMounts(cmds []string,resources exports.GObject) ([]string, []JOB.MountPoint){
	 action := make([]string,0,len(cmds))
	 mounts := []JOB.MountPoint{}

	 maps   := make(map[string]int)

	 for _,value := range(cmds){
	 	name,subname,extra := fetchCmdResource(value)
	 	if rsc,ok :=resources[name].(exports.GObject);ok && len(name) > 0 {
	 		path,_    := rsc["path"].(string)
	 		rpath,_   := rsc["rpath"].(string)
	 		subpath   := ""
			access    := int(safeToNumber(rsc["access"]))

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
				 access     := int(safeToNumber(rsc["access"]))

				 mounts,_ = tryResourceMounts(name,path,rpath,access,"","",maps,mounts)

				 if subresource,_ := rsc["subResource"].(exports.GObject);len(subresource)>0{
				 	for subname,subpath :=range(subresource) {
				 		mounts,_=tryResourceMounts(name,path,rpath,access,subname,safeToString(subpath),maps,mounts)
					}
				 }

			 }
		 }
	 }
	 return action,mounts
}

func checkEnableSSH(run*models.Run,job * JOB.Job) bool {
	  ports := []models.UserEndpoint{}
	  if  err := run.Endpoints.Fetch(&ports) ; err == nil  {
	  	 for _,v := range(ports) {
	  	 	if v.Name == exports.AILAB_SYS_ENDPOINT_SSH{
	  	 		job.Envs["SSH_PASSWD"] = utils.GenerateRandomStr(6)

	  	 		return true
		    }
	     }
	  }
	  return false
}

func checkSpeciaEnvs(run*models.Run,job *JOB.Job)   {

	 //@todo: create all job in default namespace
	 job.Namespace = "default"

	 if !strings.HasSuffix(job.Cmd[0],"_launcher"){// cmd not launcher , start target container directly !!!
		return
	 }
	 preStart := "01.init_user.sh"
	 if exports.IsJobRunWithCloud(run.Flags) {//does not need any local devices
	 	 job.Quota.Request.Device=JOB.Device{}
		 job.Quota.Limit.Device  =JOB.Device{}
	 }
	 if job.Quota.Request.Device.ComputeType == "huawei_npu" {
	 	 job.MountPoints = append(job.MountPoints,JOB.MountPoint{
			 Path:          "file:///usr/local/Ascend/driver",
			 ContainerPath: "/usr/local/Ascend/driver",
			 ReadOnly:      true,
		 },JOB.MountPoint{
		     Path:          "file:///usr/local/Ascend/add-ons",
		     ContainerPath: "/usr/local/Ascend/add-ons",
		     ReadOnly:      true,
	     })
	 	 preStart += " 02.setup_mindspore.sh"
	 }
	 if checkEnableSSH(run,job) {
	 	 preStart += " 03.setup_ssh.sh"
	 }
	 job.PreStartScripts=preStart
}

func checkAILabEnvs(run*models.Run,envs map[string]string ){
    envs[exports.AILAB_ENV_ADDR]   = fmt.Sprintf("http://ai-lab.default:%d%s",configs.GetAppConfig().Port,exports.AILAB_API_VERSION)
    envs[exports.AILAB_ENV_LAB_ID] = fmt.Sprintf("%d",run.LabId)
    envs[exports.AILAB_ENV_CLUSTER_ID] = configs.GetAppConfig().ClusterId
    envs[exports.AILAB_ENV_USER_TOKEN] = run.Token
}

func SubmitJob(run*models.Run) (int, APIError) {

	 url  := configs.GetAppConfig().Resources.Jobsched+"/jobs"

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
	 job.Cmd,job.MountPoints = CheckResourceMounts(job.Cmd,resource)
	 if job.Envs == nil{
	 	job.Envs = make(map[string]string)
	 }
	 checkAILabEnvs(run,job.Envs)
	 //@todo:  add pre-start scripts ???
	 checkSpeciaEnvs(run, job)
	 resp := &JOB.CreateJobRsp{}

	 err  := Request(url,"POST",nil,job,resp)
	 if err == nil {// no error occur
	 	   return translateJobStatus(resp.JobState.Status),nil
	 }else if err.Errno() == JOB.ERR_CODE_RESOURCE_EXIST{// create job already exists
        //@todo: will return exists status also ???
	 	status := translateJobStatus(resp.JobState.Status)
	 	if exports.IsRunStatusActive(status) {// still active yet
           return status,nil
		}
        if run.EnableResume(){//try delete job and resume again
         	err = DeleteJob(run.RunId)
         	logger.Warnf("try submit resumable job:%s already exists , delete first ...",run.RunId)
         	if err != nil {
         		return exports.AILAB_RUN_STATUS_INVALID,err
			}
			//submit again
			err = Request(url,"POST",nil,job,resp)
			if err == nil {
				return translateJobStatus(resp.JobState.Status),nil
			}else if err.Errno() == JOB.ERR_CODE_INVALID_PARAM{
				return exports.AILAB_RUN_STATUS_FAIL,err
			}else{
				return exports.AILAB_RUN_STATUS_INVALID,err
			}
		 }else{//should not change run status here
		 	logger.Warnf("try submit job:%s already exists!",run.RunId)
            return exports.AILAB_RUN_STATUS_FAIL,err
		 }
	 }else if err.Errno() == JOB.ERR_CODE_INVALID_PARAM{
	 	 return exports.AILAB_RUN_STATUS_FAIL,err
	 }else{
	 	 return exports.AILAB_RUN_STATUS_INVALID,err
	 }
}

func KillJob(run*models.Run) (int,APIError) {
	 if err := DeleteJob(run.RunId);err == nil{
	 	return exports.AILAB_RUN_STATUS_ABORT,nil
	 }else{
	 	return exports.AILAB_RUN_STATUS_INVALID,err
	 }
}

func DeleteJob(runId string) APIError{
	 url  := fmt.Sprintf("%s/jobs/%s",configs.GetAppConfig().Resources.Jobsched,runId)
	 err  := Request(url,"DELETE",nil,nil,nil)
	 if err == nil || err.Errno() == JOB.ERR_CODE_RESOURCE_NOT_EXIST {//should never error
		return nil
	 }else{
		return err
	 }
}

func GetJobLogs(runId string,pageNum string) (interface{},APIError){
	 url := fmt.Sprintf("%s/logs/%s?pageNum=%s",configs.GetAppConfig().Resources.Jobsched,runId,pageNum)
	 var result interface{}
	 err := Request(url,"GET",nil,nil,&result)
	 return result,err
}

func  SyncJobStatus(runId string,statusFrom int, statusTo int,err APIError) APIError{
	 if statusTo == exports.AILAB_RUN_STATUS_INVALID {//cancel this action
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
		  if statusTo == exports.AILAB_RUN_STATUS_INVALID {
		  	logger.Warnf("receive unknown job state from mq:%s",string(event.Message().Body))
		  }else{
		  	logger.Info("receive mq message:%s",string(event.Message().Body))
		  	return models.ChangeJobStatus(jobs.JobId,exports.AILAB_RUN_STATUS_INVALID,statusTo,jobs.JobState.Msg)
		  }

	  }
	  return nil
}

