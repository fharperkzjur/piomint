
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

type EngineResourceSrv struct{

}
//cannot error
func (d EngineResourceSrv) PrepareResource(runId string,  resource exports.GObject) (interface{},APIError){

	return nil,exports.NotImplementError("EngineResourceSrv")
}

// should never error
func (d EngineResourceSrv) CompleteResource(runId string,resource exports.GObject,commitOrCancel bool) APIError {
    //@todo: engine no need to ref ???
	return nil
}

func ValidateEngineUrl(engine string,arch string)(string, string, APIError){
	 if engine == exports.AILAB_ENGINE_DEFAULT {//use internal init-container engine
	 	return engine,arch,nil
	 }else if engineId,err := strconv.ParseUint(engine,10,64);err == nil && engineId > 0 {//resolve id to full name by apharbor
        return getAPHarborImageUrl(engineId,arch)
     }else if engine[0] == '#' {//@todo: validate user request name ???
     	return engine[1:],arch,nil
     }else if paths := strings.SplitN(engine,"/",2);len(paths) == 2 && strings.ContainsAny(paths[0],".:"){//already full name
     	return engine,arch,nil
     }else {// resolve relative name:tag to full name by apharbor
        return getAPHarborImageUrlByName(engine,arch)
     }
}

func getAPHarborImageUrl(id uint64,arch string) (string,string,APIError){
	 url := fmt.Sprintf("%s/images/imageVersion/%d",configs.GetAppConfig().Resources.ApHarbor)
	 type ImageVersionInfo struct{
		 ImageFullPath string `json:"imageFullPath"`
		 Arch          string   //@todo:  no arch information yet ???
	 }
	 image := &ImageVersionInfo{}
	 if err := Request(url,"GET",nil,nil,image);err != nil {
	 	return "","",nil
	 }
	 if image.Arch == "" {//@todo:  should join this two arch here ?
	 	image.Arch=arch
	 }
	 return image.ImageFullPath,image.Arch,nil
}
func getAPHarborImageUrlByName(name string,arch string) (string,string,APIError){
	url := fmt.Sprintf("%s/imagesByName?name=%s",configs.GetAppConfig().Resources.ApHarbor,name)
	type ImageVersionInfo struct{
		ImageFullPath string `json:"imageFullPath"`
		Arch          string   //@todo:  no arch information yet ???
	}
	image := &ImageVersionInfo{}
	if err := Request(url,"GET",nil,nil,image);err != nil {
		return "","",nil
	}
	if image.Arch == "" {//@todo:  should join this two arch here ?
		image.Arch=arch
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
