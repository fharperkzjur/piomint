
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


func CreateLinkWith(ctx string,refer string) APIError{
	return execDBTransaction(func(tx*gorm.DB)APIError{
		   // must refer to an exists completed run
           mlrun,err := getBasicMLRunInfo(tx,0,refer)
           if err == nil &&  !mlrun.HasCompleteOK() {
           	  err = exports.RaiseAPIError(exports.AILAB_INVALID_RUN_STATUS)
		   }
		   if err == nil {
		   	  err = wrapDBQueryError(tx.FirstOrCreate(&Link{
				  Ctx:       ctx,
				  Refer:     refer,
			  }))
		   }
		   return err
	})
}

func DeleteLinkWith(ctx string,refer string) APIError{

	return execDBTransaction(func(tx*gorm.DB)APIError{

		mlrun,err := getBasicMLRunInfo(tx.Unscoped(),0,refer)
		if err != nil {
			if err.Errno() == exports.AILAB_NOT_FOUND { //do nothing if refer not exists
				err=nil
			}
			return err
		}
		err = wrapDBUpdateError(tx.Delete(&Link{},"ctx=? and refer=?",ctx,refer),1)

		if err == nil && mlrun.NeedClean() {
            return tryDiscardRun(tx,refer)
		}else if err != nil && err.Errno() == exports.AILAB_DB_UPDATE_UNEXPECT{// no refer exists
			return nil
		}else{
			return err
		}

	})

}

func tryDiscardRun(tx*gorm.DB,runId string) APIError {
	err := wrapDBQueryError(tx.First(&Link{},"refer=?",runId))
	if err == nil {// do nothing if link exists !
		return nil
	}
	if err.Errno() != exports.AILAB_NOT_FOUND {
		return err
	}
	//auto discard when no refered clean runs !
	err = wrapDBUpdateError( tx.Table("runs").Unscoped().Where("run_id=? and status=?",runId,exports.RUN_STATUS_PRE_CLEAN).
		 UpdateColumn("status",exports.RUN_STATUS_DISCARD),1)
	if err == nil {
		err = doLogDiscardRun(tx,runId)
	}else if err.Errno() == exports.AILAB_DB_UPDATE_UNEXPECT{
		err = nil
	}
	return err
}

func DiscardRun(runId string) APIError{
	return execDBTransaction(func(tx*gorm.DB)APIError{
		mlrun,err := getBasicMLRunInfo(tx.Unscoped(),0,runId)
		if err != nil {
			if err.Errno() == exports.AILAB_NOT_FOUND { //do nothing if refer not exists
				err=nil
			}
			return err
		}
		if !mlrun.NeedClean(){
			return nil
		}
		return tryDiscardRun(tx,runId)
	})
}

func DisposeRun(run* Run)APIError{
     // deleted run output storage & then delete from db
	 err := deleteStg(run.Output)
	 if err!= nil {
	 	return err
	 }
	 return wrapDBUpdateError( db.Unscoped().Delete(run),1)
}

