
package services

import "github.com/apulis/bmod/ai-lab-backend/pkg/exports"

type DatasetResourceSrv struct{

}
//cannot error
func (d DatasetResourceSrv) PrepareResource(runId string,  resource exports.GObject) (interface{},APIError){



	return nil,exports.NotImplementError("DatasetResourceSrv")
}

// should never error
func (d DatasetResourceSrv) CompleteResource(runId string,resource exports.GObject,commitOrCancel bool) APIError {



	return exports.NotImplementError("DatasetResourceSrv")
}
