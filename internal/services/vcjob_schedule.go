
package services

import (
	"encoding/base64"
	"fmt"
	"github.com/apulis/bmod/ai-lab-backend/internal/configs"
	"github.com/apulis/bmod/ai-lab-backend/internal/models"
	"github.com/apulis/bmod/ai-lab-backend/pkg/exports"
	JOB "github.com/apulis/go-business/pkg/jobscheduler"
	"strings"
)

func checkEnableUserEndpoints(run*models.Run, container * JOB.Container) (helper string) {
	    endpoints :=  []models.UserEndpoint{}
	    run.Endpoints.Fetch(&endpoints)
		for idx,v := range(endpoints) {
			if v.Name == exports.AILAB_SYS_ENDPOINT_SSH {// force ssh to be nodePort service
				if len(v.SecureKey) > 0 {
					sk , _ := base64.StdEncoding.DecodeString(v.SecureKey)
					container.Envs["SSH_PASSWD"] = string(sk)
				}
				helper += " 03.setup_ssh.sh"
				container.Ports[idx].ServiceType = JOB.ServiceTypeNodePort
				continue
			}
			if v.Name == exports.AILAB_SYS_ENDPOINT_NNI{
				container.Envs["AILAB_NNI_ENDPOINT"] = fmt.Sprintf("%s:%d",v.ServiceName,v.Port)
				helper += " 04.patch_nni.sh"
			}
			container.Ports[idx].ServiceType = JOB.ServiceTypeClusterIP
		}
		return
}

func checkAppUsage(run*models.Run,task* JOB.VcJobTask)   {

	//@todo: affinity support ???
	if exports.IsJobNeedAffinity(run.Flags) {
		//task.Affinity = run.Parent
	}
	if run.Image == exports.AILAB_ENGINE_DEFAULT {// use init-container as target logic run container
		task.Container.ImageName = configs.GetAppConfig().InitToolImage
		if run.JobType == exports.AILAB_RUN_SCRATCH {//only this `scratch` JOB need docker support & root user
			task.Container.MountPoints = append(task.Container.MountPoints,JOB.MountPoint{
				Path:          "file:///var/run/docker.sock",
				ContainerPath: "/var/run/docker.sock",
				ReadOnly:      false,
			})
			task.Container.Envs[JOB.ENV_KEY_RUN_AS_ROOT]=JOB.ENV_VALUE_RUN_AS_ROOT
		}
		return
	}
	if !strings.HasSuffix(task.Container.Cmd[0],"_launcher"){// cmd not launcher , start target container directly !!!
		return
	}
	preStart := "01.init_user.sh"
	if exports.IsJobRunWithCloud(run.Flags) {//does not need any local devices
		task.Container.ResourceQuota.Request.Device=JOB.Device{}
		task.Container.ResourceQuota.Limit.Device  =JOB.Device{}
	}
	if task.Container.ResourceQuota.Request.Device.ComputeType == "huawei_npu" {//support for huawei driver mount to docker
		task.Container.MountPoints = append(task.Container.MountPoints,JOB.MountPoint{
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

	preStart += checkEnableUserEndpoints(run,task.Container)
	task.InitContainer=&JOB.Container{
		ContainerName: "init-tools",
		ImageName:     configs.GetAppConfig().InitToolImage,
		Envs:          make(map[string]string,0),
		Cmd:           []string{"bash", "-c", "cp -r /code/prestart/* /prestart && cp -r /code/start/* /start/ && cp -r /dlts-init/* /dlts-runtime/"},
		MountPoints:   []JOB.MountPoint{
			{
				Path:"emptydir://prestart",
				ContainerPath:  "/prestart",
			},
			{
				Path:"emptydir://start",
				ContainerPath:  "/start",
			},
			{
				Path:"emptydir://dlts-runtime",
				ContainerPath: "/dlts-runtime",
			},
		},
	}
	task.Container.Envs["JOB_CMD"]=strings.Join(task.Container.Cmd," ")
	task.Container.Envs["PRESTART_SCRIPTS"] = preStart
	task.Container.Cmd=[]string{"bash","-c","/prestart/prestart.sh && /start/start.sh"}
	for _,v := range(task.InitContainer.MountPoints) {
		v.ReadOnly=true
		task.Container.MountPoints=append(task.Container.MountPoints,v)
	}
}

func TagAILabEnvs(run*models.Run,envs map[string]string ){
	envs[exports.AILAB_ENV_ADDR]   = fmt.Sprintf("http://ai-lab.default:%d%s",configs.GetAppConfig().Port,exports.AILAB_API_VERSION)
	envs[exports.AILAB_ENV_LAB_ID] = fmt.Sprintf("%d",run.LabId)
	envs[exports.AILAB_ENV_CLUSTER_ID] = configs.GetAppConfig().ClusterId
	envs[exports.AILAB_ENV_USER_TOKEN] = run.Token
}

func CloneTaskEnv(envs map[string]string)map[string]string{
	 newEnvs := make(map[string]string)
	 for k,v := range(envs) {
	 	newEnvs[k]=v
	 }
	 return newEnvs
}

func CreateVcWorkerTask(task*JOB.VcJobTask, node int,compactMaster bool ) []JOB.VcJobTask {

	 task.Container.Envs["VCJOB_TASK_NAME"] = task.TaskName

	 if task.ArchType == "" {//@todo: here must specify arch type ???
	 	task.ArchType="amd64"
	 }

	 worker := JOB.VcJobTask{
		 TaskName:      "worker",
		 Replicas:      node,
		 ArchType:      task.ArchType,
	 }
	 worker.Container = &JOB.Container{
		 ContainerName: task.Container.ContainerName,
		 ImageName:     task.Container.ImageName,
		 Cmd:           task.Container.Cmd,
		 MountPoints:   task.Container.MountPoints,
		 ResourceQuota: task.Container.ResourceQuota,
		 Envs:          CloneTaskEnv(task.Container.Envs),
		 Ports:         nil, // worker node does not need any endpoints
	 }
	 if compactMaster {
	 	worker.Replicas --
	 }
	 delete(worker.Container.Envs,"SSH_PASSWD")
	 delete(worker.Container.Envs,"AILAB_NNI_ENDPOINT")
	 worker.Container.Envs["VCJOB_TASK_NAME"] = worker.TaskName

	 if task.InitContainer != nil {//copy init-container also
	 	 task.InitContainer.Envs["VCJOB_TASK_NAME"] = task.TaskName
	 	 worker.InitContainer=&JOB.Container{
		     ContainerName: task.InitContainer.ContainerName,
		     ImageName:     task.InitContainer.ImageName,
		     Cmd:           task.InitContainer.Cmd,
		     MountPoints:   task.InitContainer.MountPoints,
		     ResourceQuota: task.InitContainer.ResourceQuota,
		     Envs:          CloneTaskEnv(task.InitContainer.Envs),
		     Ports:         nil,
	     }

	     worker.InitContainer.Envs["VCJOB_TASK_NAME"]=worker.TaskName
	 }

	 if !compactMaster {//purge device info from master
          task.Container.ResourceQuota.Request.Device = JOB.Device{}
		  task.Container.ResourceQuota.Limit.Device = JOB.Device{}

		  if task.InitContainer != nil {
			  task.InitContainer.ResourceQuota.Request.Device = JOB.Device{}
			  task.InitContainer.ResourceQuota.Limit.Device = JOB.Device{}
		  }
	 }

	 tasks := []JOB.VcJobTask{
	 	*task,
	 	worker,
	 }
	 return tasks
}

func CreateVcJobTask(run*models.Run) ([]JOB.VcJobTask,APIError){
	quota := &models.UserResourceQuota{}
	if err := run.Quota.Fetch(&quota);err != nil {
		return nil,exports.RaiseAPIError(exports.AILAB_LOGIC_ERROR,"invalid quota specified by user !!!")
	}
	task :=  JOB.VcJobTask{
		TaskName:      "master",
		Replicas:      1,
		ArchType:      run.Arch,
	}
	container := &JOB.Container{
		ContainerName: run.RunId,
		ImageName:     run.Image,
		Cmd:           make([]string,0),
		Envs:          make(map[string]string,0),
		ResourceQuota: quota.ResourceQuota,
	}
	run.Cmd.Fetch(&container.Cmd)
	run.Envs.Fetch(&container.Envs)
	run.Endpoints.Fetch(&container.Ports)

	resource := exports.GObject{}
	run.Resource.Fetch(&resource)
	container.Cmd,container.MountPoints = CheckResourceMounts(container.Cmd,resource)
	checkAILabEnvs(run,container.Envs)
	task.Container=container

	checkAppUsage(run,&task)
	if quota.Node < 2 {
		return []JOB.VcJobTask{task},nil
	}
	return  CreateVcWorkerTask(&task,quota.Node,exports.IsJobDistributeCompactMaster(run.Flags)),nil
}

func submitJobInternal(run*models.Run, url string,job interface{}) (int,APIError){

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

func SubmitVcJob(run*models.Run, tasks []JOB.VcJobTask) (int, APIError) {
	vcjob := &JOB.CreateDistributedJobReq{}
	vcjob.JobBase=JOB.JobBase{
		ModId:       exports.AILAB_MODULE_ID,
		JobId:       run.RunId,
		ResType:     JOB.RESOURCE_TYPE_DISTRIBUTED_JOB,
		Owner:       run.Creator,
		UserGroupId: 0,  //@todo:  retrive actually user group ???
		Namespace:   run.Namespace,
	  }
	 vcjob.Tasks=tasks
	 url  := configs.GetAppConfig().Resources.Jobsched+"/distributed-jobs"
	 return submitJobInternal(run,url,vcjob)
}
func SubmitSingleJob(run*models.Run,task * JOB.VcJobTask) (int,APIError){
	
	job := &JOB.Job{
		ModId:           exports.AILAB_MODULE_ID,
		JobId:           run.RunId,
		Owner:           run.Creator,
		UserGroupId:     0, //@todo:  retrive actually user group ???
		ResType:         JOB.RESOURCE_TYPE_JOB,
		ImageName:       task.Container.ImageName,
		Cmd:             task.Container.Cmd,
		Namespace:       run.Namespace,
		Envs:            task.Container.Envs,
		MountPoints:     task.Container.MountPoints,
		ArchType:        task.ArchType,
		Quota:           task.Container.ResourceQuota,
		PreStartScripts: "", //@mark:  donot use this parameters any more !!!
		InitContainer:   task.InitContainer,
	}

	url  := configs.GetAppConfig().Resources.Jobsched+"/jobs"
	return submitJobInternal(run,url,job)
}

func SubmitJobV2(run*models.Run) (int, APIError) {
	tasks , err := CreateVcJobTask(run)
	if err != nil {
		return exports.AILAB_RUN_STATUS_FAIL,err
	}
	//@todo:  `default` ???
	run.Namespace="default"
	if len(tasks) > 1 {
		return SubmitVcJob(run,tasks)
	}else {
		return SubmitSingleJob(run,&tasks[0])
	}
}
