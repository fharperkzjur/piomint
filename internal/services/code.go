
package services

import (
	"fmt"
	"github.com/apulis/bmod/ai-lab-backend/internal/configs"
	"github.com/apulis/bmod/ai-lab-backend/pkg/exports"
	"strconv"
	"strings"
)

type CodeResourceSrv struct{

}
//cannot error
func (d CodeResourceSrv) PrepareResource(runId string,  resource exports.GObject) (interface{},APIError){
	// no need to ref &unref code now , because project manage the whole repo lifecycle this time

	return nil,nil
}

// should never error
func (d CodeResourceSrv) CompleteResource(runId string,resource exports.GObject,commitOrCancel bool) APIError {
	// no need to ref &unref code now , because project manage the whole repo lifecycle this time

	return nil
}

type HarborResourceSrv struct{

}
//cannot error
func (d HarborResourceSrv) PrepareResource(runId string,  resource exports.GObject) (interface{},APIError){

	return nil,exports.NotImplementError("EngineResourceSrv")
}

// should never error
func (d HarborResourceSrv) CompleteResource(runId string,resource exports.GObject,commitOrCancel bool) APIError {
    //@todo: engine no need to ref ???
	return nil
}

func ValidateEngineUrl(token string, engine string,arch string)(string, string, APIError){
	 if engine == "" {
	 	return "","",exports.ParameterError("empty engine name !!!")
	 }else if engine == exports.AILAB_ENGINE_DEFAULT {//use internal init-container engine
	 	return engine,arch,nil
	 }else if engineId,err := strconv.ParseUint(engine,10,64);err == nil && engineId > 0 {//resolve id to full name by apharbor
        return getAPHarborImageUrl(token,engineId,arch)
     }else if engine[0] == '#' {//@todo: validate user request name ???
     	return engine[1:],arch,nil
     }else if paths := strings.SplitN(engine,"/",2);len(paths) == 2 && strings.ContainsAny(paths[0],".:"){//already full name
     	return engine,arch,nil
     }else {// resolve relative name:tag to full name by apharbor
        return getAPHarborImageUrlByName(token,engine,arch)
     }
}

func getAPHarborImageUrl(token string, id uint64,arch string) (string,string,APIError){
	 url := fmt.Sprintf("%s/images/imageVersion/%d",configs.GetAppConfig().Resources.ApHarbor,id)
	 type ImageVersion struct{
		 ImageFullPath string `json:"imageFullPath"`
		 Arch          string   //@todo:  no arch information yet ???
	 }
	 type ImageVersionRsp struct{
		 ImageVersion ImageVersion `json:"imageVersion"`
	 }
	 image := &ImageVersionRsp{}
	 if err := Request(url,"GET",map[string]string{
	 	"Authorization" : "Bearer " + token,
	 },nil,image);err != nil {
	 	return "","",err
	 }
	 if image.ImageVersion.Arch == "" {//@todo:  should join this two arch here ?
	 	image.ImageVersion.Arch=arch
	 }
	 if image.ImageVersion.ImageFullPath == "" {

	 	return "","",exports.RaiseAPIError(exports.AILAB_REMOTE_REST_ERROR,"getAPHarborImageUrl empty response !!!")
	 }
	 return image.ImageVersion.ImageFullPath,image.ImageVersion.Arch,nil
}
func getAPHarborImageUrlByName(token string,name string,arch string) (string,string,APIError){

	url := fmt.Sprintf("%s/images/getImageVersion?imageFullName=%s",configs.GetAppConfig().Resources.ApHarbor,name)
	type ImageVersion struct{
		ImageFullPath string `json:"imageFullPath"`
		Arch          string   //@todo:  no arch information yet ???
	}
	image := &ImageVersion{}
	if err := Request(url,"GET",map[string]string{
		"Authorization" : "Bearer " + token,
	},nil,image);err != nil {
		return "","",err
	}
	if image.Arch == "" {//@todo:  should join this two arch here ?
		image.Arch=arch
	}
	if image.ImageFullPath == "" {
		return "","",exports.RaiseAPIError(exports.AILAB_REMOTE_REST_ERROR,"getAPHarborImageUrlByName empty response !!!")
	}
	return image.ImageFullPath,image.Arch,nil
}

type StoreResourceSrv struct{

}

//cannot error
func (d StoreResourceSrv) PrepareResource(runId string,  resource exports.GObject) (interface{},APIError){

	if len(safeToString(resource["path"])) == 0 {
		return nil,exports.ParameterError("store resource must have valid path !!!")
	}
	return nil,nil
}

// should never error
func (d StoreResourceSrv) CompleteResource(runId string,resource exports.GObject,commitOrCancel bool) APIError {
	return nil
}
