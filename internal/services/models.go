
package services

import "github.com/apulis/bmod/ai-lab-backend/pkg/exports"

type ModelResourceSrv struct{

}
//cannot error
func (d ModelResourceSrv) PrepareResource (runId string,  resource exports.GObject) (interface{},APIError){


      return nil,exports.NotImplementError("ModelResourceSrv")
}

// should never error
func (d ModelResourceSrv) CompleteResource(runId string,resource exports.GObject,commitOrCancel bool) APIError {



	  return exports.NotImplementError("ModelResourceSrv")
}
