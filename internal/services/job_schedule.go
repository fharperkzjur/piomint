
package services

import (
	"github.com/apulis/bmod/ai-lab-backend/internal/models"
	"github.com/apulis/bmod/ai-lab-backend/pkg/exports"
)

func SubmitJob(run*models.Run) (int,APIError) {


	 return 0,exports.NotImplementError("SubmitJob")
}

func KillJob(run*models.Run) (int,APIError) {
	 return 0,exports.NotImplementError("KillJob")
}

func DeleteJob(runId string) APIError{
	 return exports.NotImplementError("DeleteJob")
}

