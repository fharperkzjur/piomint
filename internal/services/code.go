
package services

import "github.com/apulis/bmod/ai-lab-backend/pkg/exports"

type CodeResourceSrv struct{

}
//cannot error
func (d CodeResourceSrv) PrepareResource(runId string,  resource exports.GObject) (interface{},APIError){



	return nil,exports.NotImplementError("CodeResourceSrv")
}

// should never error
func (d CodeResourceSrv) CompleteResource(runId string,resource exports.GObject,commitOrCancel bool) APIError {


	return exports.NotImplementError("CodeResourceSrv")
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
