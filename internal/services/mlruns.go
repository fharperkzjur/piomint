package services

import (
	"encoding/csv"
	"encoding/json"
	"github.com/apulis/bmod/ai-lab-backend/internal/configs"
	"github.com/apulis/bmod/ai-lab-backend/internal/models"
	"github.com/apulis/bmod/ai-lab-backend/pkg/exports"
	"github.com/gin-gonic/gin"
	"io/ioutil"
	"net/http"
	"os"
	"path"
)

type APIError = exports.APIError




type MlrunResourceSrv struct{

}
//cannot error
func (d MlrunResourceSrv) PrepareResource(runId string, token string, resource exports.GObject) (interface{},APIError){

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
func (d MlrunResourceSrv) CompleteResource(runId string,token string,resource exports.GObject,commitOrCancel bool) APIError {
	refer  := safeToString(resource["id"])
	return models.DeleteLinkWith(runId,refer)
}

type MlrunOutputSrv struct{

}
//cannot error
func (d MlrunOutputSrv) PrepareResource(runId string, token string,resource exports.GObject) (interface{},APIError){

	path,err := models.EnsureLabRunStgPath(runId)
	if err != nil {
		return nil,err
	}
	return map[string]interface{}{
		"path":path,
	},nil
}

// should never error
func (d MlrunOutputSrv) CompleteResource(runId string,token string,resource exports.GObject,commitOrCancel bool) APIError {

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

	//@mark: validate engine
	var err1 APIError
	if req.Engine,req.Arch,err1 = ValidateEngineUrl(req.Token,req.Engine,req.Arch);err1 != nil {
		return nil,err1
	}
	//@mark: validate user endpoints
	if err1 = ValidateUserEndpoints(req.Endpoints);err1 != nil {
		return nil,err1
	}

	if len(req.Cmd) == 0 {
		return nil,exports.ParameterError("invalid run cmd !!!")
	}
	if  len(req.Engine) == 0{
		return nil,exports.ParameterError("invalid run engine name !!!")
	}
	if len(req.Creator) == 0 {
		return nil,exports.ParameterError("run creator cannot be empty !!!")
	}
	if  req.UserId == 0 {
		return nil,exports.ParameterError("run user id cannot be empty !!!")
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
			rsc["access"] =1
			req.OutputPath=exports.AILAB_OUTPUT_NAME
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
		}else if ty == exports.AILAB_RESOURCE_TYPE_STORE {//do nothing with `store` resource , becare to check authority before call

		}else if ty == exports.AILAB_RESOURCE_TYPE_OUTPUT{//@todo:pseudo output as a refered resource ???
			req.JobFlags |= exports.AILAB_RUN_FLAGS_NEED_REFS
			req.OutputPath = exports.AILAB_OUTPUT_NAME
			rsc["access"]  = 1
			rsc["rpath"]   = getPVCMappedPath(exports.AILAB_OUTPUT_NAME,"","")
		}else{
			if id:= safeToString(rsc["id"]);len(id) == 0{
				return nil,exports.ParameterError("invalid resource id with name:" + name)
			}else if safeToNumber(rsc["access"]) != 0{
				req.JobFlags |= exports.AILAB_RUN_FLAGS_NEED_SAVE
			}else{
				req.JobFlags |= exports.AILAB_RUN_FLAGS_NEED_REFS
			}
			if configs.GetAppConfig().Debug && safeToString(rsc["path"]) != "" {
				rsc[exports.AILAB_RESOURCE_NO_REFS]=1
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
	}else if ext == ".json" && c.Query("format") == "1" {// wrap json file to common message
		contents,err := ioutil.ReadFile(targetPath)
		if err != nil {
			return exports.RaiseAPIError(exports.AILAB_OS_IO_ERROR,err.Error())
		}
		var jst  interface{}
		err = json.Unmarshal(contents,&jst)
		if err != nil {
			return exports.RaiseAPIError(exports.AILAB_INVALID_FORMAT,"invalid json data")
		}
		c.JSON(http.StatusOK,exports.CommResponse{
			Data: jst,
		})
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