package services

import (
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github.com/apulis/bmod/ai-lab-backend/internal/configs"
	"github.com/apulis/bmod/ai-lab-backend/internal/models"
	"github.com/apulis/bmod/ai-lab-backend/pkg/exports"
	JOB "github.com/apulis/go-business/pkg/jobscheduler"
	"github.com/gin-gonic/gin"
	"net/http"
	"os"
	"path"
	"strings"
)

type APIError = exports.APIError

func genEndpointsUrl(name  ,service ,namespace string,port int) string{
	 url := configs.GetAppConfig().GatewayUrl
	 jsonInfo := map[string]interface{}{
	 	"service":service + "." + namespace + "." + ".svc.cluster.local",
	 	"port":port,
	 }
	 vhost ,_:= json.Marshal(jsonInfo)

	 return fmt.Sprintf("%s/endpoints/%s/%s/",url,name, base64.StdEncoding.EncodeToString(vhost))
}


func GetEndpointUrl(mlrun*models.BasicMLRunContext,name string) (interface{},APIError){
	 if mlrun.StatusIsRunning() {
	 	 if mlrun.Endpoints == nil {
	 	 	return nil,exports.RaiseAPIError(exports.AILAB_LOGIC_ERROR,"no endpoints created for run:"+mlrun.RunId)
	     }
	     endpoints := []JOB.ContainerPort{}
	     if err:=mlrun.Endpoints.Fetch(&endpoints);err != nil {
	     	return nil,exports.RaiseAPIError(exports.AILAB_LOGIC_ERROR,"invalid endpoints data formats for run:"+mlrun.RunId)
	     }
	     for _,v := range(endpoints) {
	     	if strings.HasPrefix(v.ServiceName,name) {

	     		return genEndpointsUrl(name,v.ServiceName,mlrun.Namespace,v.Port),nil
	        }
	     }
	     return nil,exports.NotFoundError()
	 }else{
		 return "",exports.RaiseReqWouldBlock("wait to starting job ...")
	 }
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
	if  len(req.Engine) == 0{
		return nil,exports.ParameterError("invalid run engine name !!!")
	}
	if len(req.Creator) == 0 {
		return nil,exports.ParameterError("run creator cannot be empty !!!")
	}
	if  len(req.Name) == 0 {
		return nil,exports.ParameterError("run name cannot be empty !!!")
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
		if name == exports.AILAB_OUTPUT_NAME {
			rsc["id"]     ="*"
			rsc["access"] =1
			req.OutputPath="*"
		}
	}
	for name,v := range(req.Resource){
		if len(name) == 0 {
			return nil,exports.ParameterError("resource name cannot be empty !!!")
		}
		rsc,_ := v.(exports.GObject)
		if rsc == nil {
			return nil,exports.ParameterError("invalid resource definition with name:"+name)
		}
		ty,_ := rsc["type"].(string)
		if len(ty) == 0 { ty = name }
		if ty[0] == '#' {
			var err APIError
			rsc["id"],rsc["path"],err = models.RefParentResource(parent,ty[1:])
			if err != nil {
				return nil,err
			}
		}else if ty != exports.AILAB_OUTPUT_NAME {

			if id:= safeToString(rsc["id"]);len(id) == 0{
				return nil,exports.ParameterError("invalid resource id with name:" + name)
			}else if safeToNumber(rsc["access"]) != 0{
				req.JobFlags |= exports.AILAB_RUN_FLAGS_NEED_SAVE
			}else{
				req.JobFlags |= exports.AILAB_RUN_FLAGS_NEED_REFS
			}
		}else{//@todo:  pseudo output as a refered resource ???
			req.JobFlags |= exports.AILAB_RUN_FLAGS_NEED_REFS
			req.OutputPath = "*"
			rsc["access"]  = 1
			rsc["id"]      = "*"
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

func ListRunFiles(labId uint64 ,runId string,prefix string)(interface{},APIError){
	path,err := models.GetLabRunStoragePath(labId,runId)
	if err != nil {
		return nil,err
	}
	return models.ListPathFiles(path,prefix)
}

func ServeFile(labId uint64,runId string,prefix string,c*gin.Context) APIError{
	targetPath,err := models.GetLabRunStoragePath(labId,runId)
	if err != nil {
		return err
	}
	targetPath,err = models.GetRealFilePath(targetPath,prefix)
	if err != nil {
		return err
	}
	ext := path.Ext(targetPath)
	if ext == ".csv" && c.Query("format") == "1"{// automatically parse csv file into json object
		fs, err := os.Open(targetPath)
		if err != nil {
			return exports.RaiseAPIError(exports.AILAB_OS_IO_ERROR,err.Error())
		}
		defer fs.Close()
		r1 := csv.NewReader(fs)
		content, err := r1.ReadAll()
		if err != nil {
			return exports.RaiseAPIError(exports.AILAB_OS_IO_ERROR,err.Error())
		}
		c.JSON(http.StatusOK,content)
	}else{
		c.Writer.WriteHeader(http.StatusOK)
		c.Writer.Header().Add("Content-Type", MapMimeContentType(ext))
		c.File(targetPath)
	}
	return nil

}

func MapMimeContentType(ext string) string{
	switch(ext){
	case ".csv": return "text/plain"
	case ".txt": return "text/plain"
	case ".htm": return "text/html"
	case ".html":return "text/html"
	case ".css": return "text/css"
	case ".xml": return "text/xml"
	case ".json":return "application/json"
	case ".js":  return "application/x-javascript"
	case ".png": return "image/png"
	case ".gif": return "image/gif"
	case ".jpeg":return "image/jpeg"
	case ".jpg": return "image/jpeg"
	case ".ico": return "image/x-icon"
	case ".bmp": return "image/x-ms-bmp"
	default:     return "application/octet-stream"
	}
}