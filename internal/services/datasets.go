
package services

import "github.com/apulis/bmod/ai-lab-backend/pkg/exports"

type DatasetResourceSrv struct{

}
//cannot error
func (d DatasetResourceSrv) RefResource(runId string,  resourceId string ,versionId string) (interface{},APIError){



	return nil,exports.NotImplementError("DatasetResourceSrv")
}

// should never error
func (d DatasetResourceSrv) UnRefResource(runId string,resourceId string, versionId string) APIError {



	return exports.NotImplementError("DatasetResourceSrv")
}
