
package models

import (
	"fmt"
	"github.com/apulis/bmod/ai-lab-backend/pkg/exports"
)

func allocateLabStg(lab* Lab) (APIError){

      lab.Location = fmt.Sprintf("%s/%s/%s/%d",exports.AILAB_STORAGE_ROOT,
      	  lab.Namespace,lab.App,lab.ID)

      return nil
}

func allocateLabRunStg(run *Run,mlrun * BasicMLRunContext) (APIError) {

      if run.Output == "*" {
      	 run.Output = fmt.Sprintf("%s/%s",mlrun.Location,run.RunId)
      	 //@todo: ensure path exists here ???
	  }else if len(run.Output) > 0 {
	  	 return exports.ParameterError("Cannot specify job output storage by user!")
	  }
	  return nil
}

func deleteStg(output string) (APIError){
	 if len(output) == 0{
	 	return nil
	 }
	 return exports.NotImplementError("deleteStg")
}

