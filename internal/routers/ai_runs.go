package routers

import (
	"github.com/apulis/bmod/ai-lab-backend/internal/configs"
	"github.com/apulis/bmod/ai-lab-backend/internal/models"
	"github.com/apulis/bmod/ai-lab-backend/internal/services"
	"github.com/apulis/bmod/ai-lab-backend/pkg/exports"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
	"strconv"
)

func AddGroupTraining(r*gin.Engine){

	group := r.Group(exports.AILAB_API_VERSION + "/labs/:lab")

	group.POST("/runs", wrapper(submitLabRun))
	group.POST("/runs/:runId/evaluates", wrapper(submitLabEvaluate))
	group.POST("/code-labs",wrapper(submitCodeLabRun))
    // operates on lab runs : open_visual/close_visual  , pause|resume|kill|stop
	group.POST("/runs/:runId", wrapper(postLabRuns))
    // support train&evaluate job list
	group.GET("/runs", wrapper(getAllLabRuns))
	group.GET("/runs/:runId",wrapper(queryLabRun))

	group.GET("runs/:runId/endpoints",wrapper(queryLabRunEndpoints))
	group.POST("runs/:runId/endpoints",wrapper(createLabRunEndPoint))
	group.DELETE("runs/:runId/endpoints/:name",wrapper(deleteLabRunEndPoint))

	group.GET("/stats",wrapper(queryLabRunStats))
	group.GET("/real-stats",wrapper(queryLabRunRealStats))

	group.GET( "/runs/:runId/files",   wrapper(listLabRunFiles))
	group.GET("/runs/:runId/logs",     wrapper(viewJobLogs))

	group.GET("/runs/:runId/fetch-logs",fetchLabRunLogs)
	group.GET("/runs/:runId/view",      viewLabRunFiles)

	group.DELETE("/runs/:runId", wrapper(delLabRun))
    // following interfce should only be called by admin role users
	group = r.Group(exports.AILAB_API_VERSION +"/runs")
	group.GET("",wrapper(sysGetAllLabRuns))
	group.GET("/:runId",wrapper(sysQueryLabRun))
	group.GET("/stats",wrapper(sysQueryLabRunStats))
	group.POST("/:runId", wrapper(sysPostLabRuns))

	group.POST("/clean",wrapper(sysCleanLabRuns))
	group.POST("/clean-stgy",wrapper(sysResetCleanStrategy))
}

func submitLabRun(c*gin.Context) (interface{},APIError){
	 labId, _ := parseLabRunId(c)
	 if labId == 0 {
	 	return nil,exports.ParameterError("create run invalid lab id")
	 }
	 req := &exports.CreateJobRequest{}
	 if err := c.ShouldBindJSON(req);err != nil{
	 	return nil,exports.ParameterError("invalid json data")
	 }
	 req.JobType = exports.AILAB_RUN_TRAINING
	 if req.UseModelArts {
	 	req.JobFlags = exports.AILAB_RUN_FLAGS_USE_CLOUD
	 }
	 req.Token=getUserToken(c)
	 return services.ReqCreateRun(labId,"",req,false,false)
}

func submitCodeLabRun(c*gin.Context)(interface{},APIError){
	labId, _ := parseLabRunId(c)
	if labId == 0 {
		return nil,exports.ParameterError("create code-lab invalid lab id")
	}
	req := &exports.CreateJobRequest{}
	if err := c.ShouldBindJSON(req);err != nil{
		return nil,exports.ParameterError("invalid json data")
	}
	req.JobType = exports.AILAB_RUN_CODE_DEVELOP
	if req.UseModelArts {
		req.JobFlags = exports.AILAB_RUN_FLAGS_USE_CLOUD
	}
	req.JobFlags |= exports.AILAB_RUN_FLAGS_SINGLETON_USER
	req.Token=getUserToken(c)
	return services.ReqCreateRun(labId,"",req,true,false)
}

func submitLabEvaluate(c*gin.Context)(interface{},APIError){
	labId,runId := parseLabRunId(c)
	if labId == 0 || len(runId) == 0 {
		return nil,exports.ParameterError("create nest run invalid lab id or run id")
	}
	req := &exports.CreateJobRequest{}
	if err := c.ShouldBindJSON(req);err != nil{
		return nil,exports.ParameterError("invalid json data")
	}
	req.JobType = exports.AILAB_RUN_EVALUATE
	if req.UseModelArts {
		req.JobFlags = exports.AILAB_RUN_FLAGS_USE_CLOUD
	}
	req.Token=getUserToken(c)
	return services.ReqCreateRun(labId,runId,req,false,false)
}

func registerLabRun(labId uint64, runId string,req *exports.CreateJobRequest) (interface{},APIError){
	req.JobType  = exports.AILAB_RUN_MODEL_REGISTER
	req.JobFlags = exports.AILAB_RUN_FLAGS_SINGLE_INSTANCE
	run, err := services.ReqCreateRun(labId,runId,req,true,false)
	if err == nil {// created new run
		return nil,exports.RaiseReqWouldBlock("wait to start model register job ...")
	}else if err.Errno() == exports.AILAB_STILL_ACTIVE {// exists old job
		return nil,exports.RaiseAPIError(exports.AILAB_SERVER_BUSY, "old register job still active ...")
	}else{
		return run,err
	}
}

func scratchLabRun(labId uint64, runId string,req *exports.CreateJobRequest) (interface{},APIError){
	req.JobType  = exports.AILAB_RUN_SCRATCH
	req.JobFlags = exports.AILAB_RUN_FLAGS_SINGLE_INSTANCE | exports.AILAB_RUN_FLAGS_SCHEDULE_AFFINITY
	req.Engine   = exports.AILAB_ENGINE_DEFAULT
	if len(runId) == 0 {
		return nil,exports.ParameterError("scratch JOB must have parent job id !!!")
	}
	run, err := services.ReqCreateRun(labId,runId,req,true,false)
	if err == nil {// created new run
		return nil,exports.RaiseReqWouldBlock("wait to start docker image scratch job ...")
	}else if err.Errno() == exports.AILAB_STILL_ACTIVE {// exists old job
		return nil,exports.RaiseAPIError(exports.AILAB_SERVER_BUSY, "old docker image scratch job still active ...")
	}else{
		return run,err
	}
}

func queryLabRun(c*gin.Context) (interface{},APIError){
	labId,runId := parseLabRunId(c)
	if labId == 0 || len(runId) == 0 {
		return nil,exports.ParameterError("queryLabRun invalid lab id or run id")
	}
	run, err := models.QueryRunDetail(runId,false,-1)
	if err == nil && run.LabId != labId {
		return nil,exports.RaiseAPIError(exports.AILAB_LOGIC_ERROR,"invalid lab id passed for runs")
	}
	return run,err
}
func queryLabRunEndpoints(c*gin.Context)(interface{},APIError){
	labId,runId := parseLabRunId(c)
	if labId == 0 || len(runId) == 0 {
		return nil,exports.ParameterError("queryLabRunEndpoints invalid lab id or run id")
	}
	return services.GetLabRunEndpoints(labId,runId)
}
func createLabRunEndPoint(c*gin.Context)(interface{},APIError){
	labId,runId := parseLabRunId(c)
	if labId == 0 || len(runId) == 0 {
		return nil,exports.ParameterError("createLabRunEndPoint invalid lab id or run id")
	}
	endpoint := &exports.ServiceEndpoint{}
	if err := c.ShouldBindJSON(endpoint);err != nil {
		return nil,exports.ParameterError("invalid json data")
	}
	if err := validateEndpointName(endpoint.Name);err != nil {
		return nil,err
	}
	return nil,services.CreateLabRunEndpoints(labId,runId,endpoint)
}
func deleteLabRunEndPoint(c*gin.Context)(interface{},APIError){
	labId,runId := parseLabRunId(c)
	if labId == 0 || len(runId) == 0 {
		return nil,exports.ParameterError("deleteLabRunEndPoint invalid lab id or run id")
	}
	name := c.Param("name")

	return nil,services.DeleteLabRunEndpoints(labId,runId,name)
}

func sysQueryLabRun(c*gin.Context)(interface{},APIError){
	_,runId := parseLabRunId(c)
	return models.QueryRunDetail(runId,true,-1)
}

func getAllLabRuns(c*gin.Context) (interface{},APIError){
	labId,_ := parseLabRunId(c)
	if labId == 0 {
		return nil,exports.ParameterError("invalid lab id")
	}
	cond,err := checkSearchCond(c,exports.QueryFilterMap{
		"jobType":"job_type",
		"parent" :"parent",
		"status" : "status",
		"creator":"creator",
	})
	if err != nil {
		return nil,err
	}
	data,err := models.ListAllLabRuns(cond,labId,true)
	return makePagedQueryResult(cond,data,err)
}
func sysGetAllLabRuns(c*gin.Context)(interface{},APIError){
	cond,err := checkSearchCond(c,exports.QueryFilterMap{
		"jobType":"job_type",
		"parent" :"parent",
		"status" : "status",
	})
	if err != nil {
		return nil,err
	}
	data,err := models.ListAllLabRuns(cond,0,true)
	return makePagedQueryResult(cond,data,err)
}

func queryLabRunStats(c*gin.Context) (interface{},APIError){
	labId,_ := parseLabRunId(c)
	return models.QueryLabStats(labId,c.Query("group"))
}

func queryLabRunRealStats(c*gin.Context) (interface{},APIError){
	labId,_ := parseLabRunId(c)
	return models.QueryLabRealStats(labId,c.Query("group"))
}

func sysQueryLabRunStats(c*gin.Context)(interface{},APIError){
    return models.QueryLabStats(0,c.Query("group"))
}
func postLabRuns(c*gin.Context)(interface{},APIError) {
	labId,runId := parseLabRunId(c)
	if labId == 0 || len(runId) == 0 {
		return nil,exports.ParameterError("invalid lab id or run id")
	}
	req := &exports.CreateJobRequest{}
	action:=c.Query("action")
	switch(c.Query("action")){
	  case "open_visual":
	  	   if err := c.ShouldBindJSON(req);err != nil {
	  	   	  return nil,exports.ParameterError("invalid json data")
		   }
		   return openLabRunVisual(labId,runId,req)
	  case "close_visual":
	  	   return models.KillNestRun(labId,runId,exports.AILAB_RUN_VISUALIZE,false)
	  case "register":
		  if err := c.ShouldBindJSON(req);err != nil {
			  return nil,exports.ParameterError("invalid json data")
		  }
		  req.Token = getUserToken(c)
		  return registerLabRun(labId,runId,req)
	  case "cancel_register":
	  	   return models.KillNestRun(labId,runId,exports.AILAB_RUN_MODEL_REGISTER,false)
	  case "scratch":
		  if err := c.ShouldBindJSON(req);err != nil {
			  return nil,exports.ParameterError("invalid json data")
		  }
		  req.Token = getUserToken(c)
		  return scratchLabRun(labId,runId,req)
	  case "cancel_scratch":
	  	   return models.KillNestRun(labId,runId,exports.AILAB_RUN_SCRATCH,false)
	  case "kill":
	  	  return  models.KillLabRun(labId,runId,false)
	  case "pause":
	  	  return nil,models.PauseLabRun(labId,runId)
	  case "resume":
	  	  return models.ResumeLabRun(labId,runId)
	  case "clean":
	  	  return models.CleanLabRun(labId,runId)
	  default:
	  	  return nil,exports.NotImplementError("unsupport mlrun action:"+action)
	}
}

func sysPostLabRuns(c*gin.Context)(interface{},APIError){
	_,runId := parseLabRunId(c)
	if len(runId) == 0 {
		return nil,exports.ParameterError("invalid run id")
	}
	action := c.Query("action")
	switch(c.Query("action")){
	case "kill":
		return models.KillLabRun(0,runId,false)
	case "pause":
		return nil,models.PauseLabRun(0,runId)
	case "resume":
		return models.ResumeLabRun(0,runId)
	case "clean":
		return models.CleanLabRun(0,runId)
	default:
		return nil,exports.NotImplementError("unsupport sys mlrun action:"+action)
	}
}

func delLabRun(c*gin.Context) (interface{},APIError){
	labId,runId := parseLabRunId(c)
	if labId == 0 || len(runId) == 0 {
		return nil,exports.ParameterError("invalid lab id or run id")
	}
	return models.DeleteLabRun(labId,runId,false)
}

func openLabRunVisual(labId uint64,runId string,req*exports.CreateJobRequest) (interface{},APIError){

	 req.JobFlags = exports.AILAB_RUN_FLAGS_SINGLE_INSTANCE | exports.AILAB_RUN_FLAGS_RESUMEABLE
	 req.JobType  = exports.AILAB_RUN_VISUALIZE
	 run, err := services.ReqCreateRun(labId,runId,req,true,false)
	 if err == nil {// created new run
	 	return nil,exports.RaiseReqWouldBlock("wait to start visual job ...")
	 }else if err.Errno() == exports.AILAB_SINGLETON_RUN_EXISTS || err.Errno() == exports.AILAB_STILL_ACTIVE{// exists old job
	 	 job := run.(*models.JobStatusChange)
	 	 run, err := models.ResumeLabRun(labId,job.RunId)
	 	 if err == nil{
	 	 	if len(req.Endpoints) == 0 {
	 	 		return nil,exports.ParameterError("openLabRunVisual must have endpoints specified !!!")
		    }
	 	 	return services.GetEndpointUrl(run,req.Endpoints[0].Name)
		 }
		 return nil,err
	 }else{
	 	return nil,err
	 }
}


func sysCleanLabRuns(c*gin.Context) (interface{},APIError){
	 req := &ReqTargetLab{}
	 if err := c.ShouldBindJSON(req);err != nil {
		return nil,exports.ParameterError("batch clean lab invalid json data")
	 }
	 return models.CleanLabRunByGroup(req.Group,req.LabID)
}

func sysResetCleanStrategy(c*gin.Context) (interface{},APIError){
	 return nil,exports.NotImplementError("sysResetCleanStrategy")
}

func parseLabRunId(c*gin.Context) (labId uint64,runId string){
	value := c.Param("lab")
	var err error
	if labId,err = strconv.ParseUint(value, 0, 64);err != nil {
		labId = 0
	}
	runId = c.Param("runId")
	return
}

func listLabRunFiles(c*gin.Context)(interface{},APIError){
	labId, runId := parseLabRunId(c)
	if labId == 0 || len(runId) == 0{
		return nil,exports.ParameterError("list run files invald labId or runId")
	}
	return services.ListRunFiles(labId,runId,c.Query("prefix"))
}

func viewLabRunFiles(c*gin.Context){

		labId, runId := parseLabRunId(c)
		if labId == 0 || len(runId) == 0{
			c.JSON(http.StatusBadRequest,exports.CommResponse{
				Code: exports.AILAB_PARAM_ERROR,
				Msg:  "list run files invald labId or runId",
			})
			return
		}
		if err := services.ServeFile(labId,runId,c.Query("prefix"),c);err != nil {
			c.JSON(http.StatusBadRequest,exports.CommResponse{
				Code: err.Errno(),
				Msg:  err.Error(),
			})
		}
}

func viewJobLogs(c*gin.Context) (interface{},APIError){
	  _,runId := parseLabRunId(c)
	  return services.GetJobLogs(runId,c.DefaultQuery("pageNum","1"))
}

func fetchLabRunLogs(c*gin.Context) {

	_,runId := parseLabRunId(c)
	forwardURL,_ := url.Parse(configs.GetAppConfig().Resources.Jobsched + "/logs/download/" + runId)
	director := func(req *http.Request) {
		req.URL.Scheme = forwardURL.Scheme
		req.URL.Host   = forwardURL.Host
		req.URL.Path   = forwardURL.Path
		req.Host = forwardURL.Host
	}
	proxy := &httputil.ReverseProxy{Director: director}
	proxy.ServeHTTP(c.Writer, c.Request)

}

var regexp_endpoint_name_validator *regexp.Regexp

func init(){
	regexp.MustCompile("^[a-z]([-a-z0-9]*[a-z0-9])?$")
}

func validateEndpointName(name string)APIError {
	if len(name) > exports.AILAB_USER_ENDPOINT_MAX_NUM{
		return exports.ParameterError("user endpoint name lenght exceed limit !!!")
	}
	if !regexp_endpoint_name_validator.MatchString(name) {
		return exports.ParameterError("user endpoint name invalid char !!!")
	}
	return nil
}
