
package services

import "github.com/apulis/bmod/ai-lab-backend/pkg/exports"

type ModelResourceSrv struct{

}
//cannot error
func (d ModelResourceSrv) RefResource(runId string,  resourceId string ,versionId string) (interface{},APIError){



      return nil,exports.NotImplementError("ModelResourceSrv")
}

// should never error
func (d ModelResourceSrv) UnRefResource(runId string,resourceId string, versionId string) APIError {



	  return exports.NotImplementError("ModelResourceSrv")
}
