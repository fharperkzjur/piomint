
package models

import (
	"github.com/apulis/bmod/ai-lab-backend/pkg/exports"
	"gorm.io/gorm"
)

type Link struct{
	Ctx       string   `gorm:"primary_key"`
	Refer     string   `gorm:"primary_key;index"`
	CreatedAt UnixTime
}


func CreateLinkWith(ctx string,refer string) (path string,err APIError){
	err = execDBTransaction(func(tx*gorm.DB,events EventsTrack)APIError{
		   // must refer to an exists completed run
           mlrun,err := getBasicMLRunInfoEx(tx,0,refer,events)
           if err == nil &&  !mlrun.StatusIsSuccess() {
           	  err = exports.RaiseAPIError(exports.AILAB_INVALID_RUN_STATUS)
		   }
		   if err == nil {
		   	  path=mlrun.Output
		   	  err = wrapDBQueryError(tx.FirstOrCreate(&Link{
				  Ctx:       ctx,
				  Refer:     refer,
			  }))
		   }
		   return err
	})
	return
}

func DeleteLinkWith(ctx string,refer string) APIError{

	return execDBTransaction(func(tx*gorm.DB,events EventsTrack)APIError{

		mlrun,err := getBasicMLRunInfoEx(tx.Unscoped().Session(&gorm.Session{}),0,refer,events)
		if err != nil {
			if err.Errno() == exports.AILAB_NOT_FOUND { //do nothing if refer not exists
				err=nil
			}
			return err
		}
		err = wrapDBUpdateError(tx.Delete(&Link{},"ctx=? and refer=?",ctx,refer),1)

		if err == nil && mlrun.ShouldDiscard() {
            return tryDiscardRun(tx,refer,events)
		}else if err != nil && err.Errno() == exports.AILAB_DB_UPDATE_UNEXPECT{// no refer exists
			return nil
		}else{
			return err
		}

	})
}
func RefParentResource(runId string,name string)(id string,path string,err APIError){

	type ParentRun struct{
		Parent  string
		JobType string
		Output  string
	}
	pr := &ParentRun{}
	for{
		err =  wrapDBQueryError(db.Select("parent,job_type,output").Model(&Run{}).First(pr,"run_id=?",runId))
		if err == nil {
			if pr.JobType == name {//find it
				if len(pr.Output) == 0 {
					err=exports.RaiseAPIError(exports.AILAB_REFER_PARENT_ERROR)
				}
				id = runId
				path = pr.Output
				return
			}
			runId= pr.Parent
		}else if err.Errno() == exports.AILAB_NOT_FOUND{
			err = exports.RaiseAPIError(exports.AILAB_REFER_PARENT_ERROR)
			return
		}else{
			return
		}
	}
}
func tryDiscardRun(tx*gorm.DB,runId string,events EventsTrack) APIError {
	err := wrapDBQueryError(tx.First(&Link{},"refer=?",runId))
	if err == nil {// do nothing if link exists !
		return nil
	}
	if err.Errno() != exports.AILAB_NOT_FOUND {
		return err
	}
	return  logDiscardRun(tx,runId,events)
}

func DisposeRun(run* Run)APIError{
     // deleted run output storage & then delete from db
	 err := deleteStg(run.Output)
	 if err!= nil {
	 	return err
	 }
	 return wrapDBUpdateError( db.Unscoped().Delete(run),1)
}

