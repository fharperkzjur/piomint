
package models

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/apulis/bmod/ai-lab-backend/pkg/exports"
	"gorm.io/gorm"
)

type Event struct{
	ID        uint64     `json:"id" gorm:"primary_key;auto_increment"`
	CreatedAt UnixTime   `json:"createdAt"`
	Type      string     `gorm:"type:varchar(32)"`// event type , should be enumarates
	Data      string     // text fields
	Extra     []byte     // compound informations for this event
}

func(evt*Event)Fetch(v interface{})error{
	return json.Unmarshal(evt.Extra,v)
}

const (
	Evt_init_run    = "init"
 	Evt_start_run   = "start"
	Evt_kill_run    = "kill"
	// check whether can clean actually
	Evt_save_run    = "save"
	// should be deleted status
	Evt_clean_run       = "clean"
	Evt_discard_run       = "discard"
	Evt_clear_lab       = "clear"
)

const (
	//Evt_clean_run_rollback = 0x1
	//Evt_clean_run_commit   = 0x2
	//Evt_clean_run_readonly = 0x4
	//Evt_clean_run_discard  = 0x8
)

const (
	Evt_clean_only            =  1
	Evt_clean_and_discard     =  2
	Evt_clean_create_rollback =  3
)

type JobEvent struct{
    runId   string
    evtent  string
    eventID uint64
}

type BackendEvents map[string]uint64
type EventsTrack *[]JobEvent

//@mark: usually within a transaction
func  LogBackendEvent(tx * gorm.DB , ty , data string,extra interface{},events EventsTrack) APIError{

	extraData :=[]byte{}
	if extra != nil {
		extraData,_=json.Marshal(extra)
	}
	evt := &Event{
		Type:      ty,
		Data:      data,
		Extra:     extraData,
	}
	err := wrapDBUpdateError(tx.Create(evt),1)
	if err == nil {
		// mark this transaction need check backend events
		*events=append(*events,JobEvent{
			runId:   data,
			evtent:  ty,
			eventID: evt.ID,
		})
	}
	return err
}

func ReadBackendEvent(event *Event , ty string,  last_processed uint64) APIError{
	return wrapDBQueryError(db.First(event," id > ? and type= ? ",last_processed,ty))
}
func RemoveBackEvent(id uint64) APIError{
	return wrapDBUpdateError(db.Delete(&Event{ID:id}),1)
}
func GetMaxBackendEventID() (eventID uint64,err APIError){
	evt := sql.NullInt64{}
	err = checkDBScanError(db.Model(&Event{}).Select("max(id)").Row().Scan(&evt))
	if err == nil {
		eventID=uint64(evt.Int64)
	}
	return
}

func logStartingRun(tx*gorm.DB, runId string,status int,events EventsTrack) APIError {
	if exports.IsRunStatusIniting(status) {
        return LogBackendEvent(tx,Evt_init_run,runId,nil,events)
	}else{//always RUN_STATUS_START
		return LogBackendEvent(tx,Evt_start_run,runId,nil,events)
	}
}

func logKillRun(tx*gorm.DB,runId string,events EventsTrack) APIError{
	return LogBackendEvent(tx,Evt_kill_run,runId,nil,events)
}

func logSaveRun(tx*gorm.DB,runId string,extra int, events EventsTrack)APIError{
	return LogBackendEvent(tx,Evt_save_run,runId,extra,events)
}

func logCleanRun(tx*gorm.DB,runId string,extra int, events EventsTrack) APIError {
	return LogBackendEvent(tx,Evt_clean_run,runId,extra,events)
}
func logDiscardRun(tx*gorm.DB,runId string,events EventsTrack) APIError{
	return LogBackendEvent(tx,Evt_discard_run,runId,nil,events)
}


//@todo:  lab runs cannot refer to another lab run ?
func logClearLab(tx*gorm.DB,labId uint64,events EventsTrack) APIError{
	return LogBackendEvent(tx,Evt_clear_lab,fmt.Sprintf("%d",labId),nil,events)
}


