
package services

import "github.com/apulis/bmod/ai-lab-backend/pkg/exports"

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
