
package services

import "github.com/apulis/bmod/ai-lab-backend/pkg/exports"

type CodeResourceSrv struct{

}
//cannot error
func (d CodeResourceSrv) RefResource(runId string,  resourceId string ,versionId string) (interface{},APIError){



	return nil,exports.NotImplementError("CodeResourceSrv")
}

// should never error
func (d CodeResourceSrv) UnRefResource(runId string,resourceId string, versionId string) APIError {


	return exports.NotImplementError("CodeResourceSrv")
}

type EngineResourceSrv struct{

}
//cannot error
func (d EngineResourceSrv) RefResource(runId string,  resourceId string ,versionId string) (interface{},APIError){

	return nil,exports.NotImplementError("EngineResourceSrv")
}

// should never error
func (d EngineResourceSrv) UnRefResource(runId string,resourceId string, versionId string) APIError {
    //@todo: engine no need to ref ???
	return nil
}
