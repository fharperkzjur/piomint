
package routers

import (
	"github.com/apulis/bmod/ai-lab-backend/internal/models"
	"github.com/apulis/bmod/ai-lab-backend/internal/services"
	"github.com/apulis/bmod/ai-lab-backend/pkg/exports"
	"github.com/gin-gonic/gin"
)

func createNNIDevExperiment(c*gin.Context) (interface{},APIError){
	labId,runId := parseLabRunId(c)
	if labId == 0 || len(runId) == 0 {
		return nil,exports.ParameterError("create nni-dev run invalid lab id or run id")
	}
	req := &exports.CreateJobRequest{}
	if err := c.ShouldBindJSON(req);err != nil{
		return nil,exports.ParameterError("invalid json data")
	}
	req.JobType = exports.AILAB_RUN_NNI_DEV
	req.Token=getUserToken(c)
	req.JobFlags = exports.AILAB_RUN_FLAGS_IDENTIFY_NAME | exports.AILAB_RUN_FLAGS_VIRTUAL_EXPERIMENT |
		exports.AILAB_RUN_FLAGS_WAIT_CHILD
	return forkChildRun(labId,runId,req)
}
func submitNNIExperimentRun(c*gin.Context)  (interface{},APIError){
	labId,_ := parseLabRunId(c)
	if labId == 0 {
		return nil,exports.ParameterError("create nni run invalid lab id ")
	}
	req := &exports.CreateJobRequest{}
	if err := c.ShouldBindJSON(req);err != nil{
		return nil,exports.ParameterError("invalid json data")
	}
	req.JobType = exports.AILAB_RUN_NNI_TRAIN
	req.JobFlags = exports.AILAB_RUN_FLAGS_WAIT_CHILD
	req.Token=getUserToken(c)
	return services.ReqCreateRun(labId,"",req,false,false)
}
func submitNNITrials(c*gin.Context) (interface{},APIError){
	labId,runId := parseLabRunId(c)
	if labId == 0 || len(runId) == 0 {
		return nil,exports.ParameterError("create nni-trail run invalid lab id or run id")
	}
	req := &exports.CreateJobRequest{}
	if err := c.ShouldBindJSON(req);err != nil{
		return nil,exports.ParameterError("invalid json data")
	}
	req.JobType  = exports.AILAB_RUN_NNI_TRIAL
	req.JobFlags = exports.AILAB_RUN_FLAGS_IDENTIFY_NAME | exports.AILAB_RUN_FLAGS_WAIT_CHILD
	req.Token=getUserToken(c)
	return forkChildRun(labId,runId,req)
}

func forkChildRun(labId uint64,runId string,req*exports.CreateJobRequest) (interface{},APIError){
	run,err := models.QueryRunDetail(runId,false,0)
	if err != nil {
		return nil,err
	}
	if labId != run.LabId {
		return nil,exports.RaiseAPIError(exports.AILAB_LOGIC_ERROR,"invalid lab id passed for runs")
	}
	if exports.IsJobNeedWaitForChilds(run.Flags) && !run.StatusIsRunning() {
		return nil,exports.RaiseAPIError(exports.AILAB_INVALID_RUN_STATUS,"only running jobs can fork sub runs !!!")
	}
	// fork child runs may inherit cmd,engine,resource,quota
	if len(req.Cmd) == 0 {
		run.Cmd.Fetch(&req.Cmd)
	}
	if len(req.Engine) == 0 {
		req.Engine = run.Image
	}
	if len(req.Arch) == 0 {
		req.Arch = run.Arch
	}
	if req.Quota == nil {
		run.Quota.Fetch(&req.Quota)
	}
	if len(req.Creator) == 0 {
		req.Creator = run.Creator
	}
	// @todo: forked child run dont inherit output path ???
	req.OutputPath = ""
	run.Resource.Fetch(&req.Resource)
	// @todo: traverse all resource and trim `path` fields when necessary ???
	checkForkedResources(req.Resource)
	if run, err := services.ReqCreateRun(labId,runId,req,false,false) ;err == nil {
		return nil,exports.RaiseReqWouldBlock("wait to start forked child job ...")
	}else if err.Errno() == exports.AILAB_SINGLETON_RUN_EXISTS {// exists old job
		job := run.(*models.JobStatusChange)
		return job.RunId,err
	}else{
		return nil,err
	}
}

func checkForkedResources(resource exports.RequestObject) {
	for name,v := range(resource){
		rsc,_ := v.(exports.GObject)
		ty,_ := rsc["type"].(string)
		if len(ty) == 0 { ty = name }
		//@mark:  forked child runs always use parent resources as `store` directory !
		rsc["type"] = exports.AILAB_RESOURCE_TYPE_STORE
	}
}
