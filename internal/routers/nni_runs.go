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
		exports.AILAB_RUN_FLAGS_WAIT_CHILD | exports.AILAB_RUN_FLAGS_RESUMEABLE
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
	run,err := models.QueryRunDetail(runId,false,0,false)
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
		quota := &models.UserResourceQuota{}
		run.Quota.Fetch(quota)
		if quota != nil {      //@todo:  cannot fork distribute vcjobs !!!
			quota.Node = 0
			req.Quota = quota
		}
	}
	if len(req.Creator) == 0 {
		req.Creator = run.Creator
	}
	if req.UserId == 0 {
		req.UserId = run.UserId
	}
	// @todo: forked child run dont inherit output path ???
	req.OutputPath = ""
	checkForkedEnvs(req,run)
	// @todo: traverse all resource and trim `path` fields when necessary ???
	checkForkedResources(req,run)
	if run, err := services.ReqCreateRun(labId,runId,req,false,false) ;err == nil {
		return run,exports.RaiseReqWouldBlock("wait to start forked child job ...")
	}else if err.Errno() == exports.AILAB_SINGLETON_RUN_EXISTS || err.Errno() == exports.AILAB_STILL_ACTIVE{// exists old job
		job := run.(*models.JobStatusChange)
		return  models.ResumeLabRun(labId,job.RunId)
	}else{
		return nil,err
	}
}

func checkForkedEnvs(req*exports.CreateJobRequest,run*models.Run) {
	run.Envs.Fetch(&req.Envs)
}

func checkForkedResources(req*exports.CreateJobRequest,run*models.Run) {

	run.Resource.Fetch(&req.Resource)

	for name,v := range(req.Resource){
		rsc,_ := v.(exports.GObject)
		ty,_ := rsc["type"].(string)
		if len(ty) == 0 { ty = name }
		//@mark:  forked child runs always use parent resources as `store` directory !
		rsc["type"] = exports.AILAB_RESOURCE_TYPE_STORE
		//@mark:  forked job use parent output path
		if ty == exports.AILAB_OUTPUT_NAME {
           if req.Envs == nil {
           	  req.Envs=make(exports.RequestTags)
		   }
		   req.Envs[exports.AILAB_ENV_OUTPUT]=rsc["rpath"].(string)
		}
	}
}
