
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
	Evt_wait_child  = "wait"
	// check whether can clean actually
	Evt_complete_run    = "complete"
	// should be deleted status
	Evt_clean_run       = "clean"
	Evt_discard_run       = "discard"
	Evt_clear_lab       = "clear"

	Evt_delete_repo     = "deleteRepo"
)

const (
	//Evt_clean_run_rollback = 0x1
	//Evt_clean_run_commit   = 0x2
	//Evt_clean_run_readonly = 0x4
	//Evt_clean_run_discard  = 0x8
)

const (
	Evt_clean_only            = 1
	Evt_clean_discard         = 2
)

type JobEvent struct{
    runId   string
    event   string
    eventID uint64
}



//type BackendEvents map[string]uint64
//type EventsTrack *[]JobEvent
type EventsTrack = * EventCollector

type EventCollector struct {
	 BackendEvents   []JobEvent //collect for backend events
	 Notifiers       []exports.NotifierData
}

func (collect*EventCollector) PushEvent(runId string,event string,eventID uint64){
      collect.BackendEvents=append(collect.BackendEvents,JobEvent{
	      runId:   runId,
	      event:   event,
	      eventID: eventID,
      })
}
func (collect*EventCollector) PushNotifier(cmd string,runId string,scope interface{},payload interface{}) {
	 collect.Notifiers=append(collect.Notifiers,exports.NotifierData{
		 Cmd:     cmd,
		 RunId:   runId,
		 Scope:   scope,
		 Payload: payload,
	 })
}

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
		//*events=append(*events,JobEvent{
		//	runId:   data,
		//	evtent:  ty,
		//	eventID: evt.ID,
		//})
		events.PushEvent(data,ty,evt.ID)
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

func logSaveRun(tx*gorm.DB,runId string, events EventsTrack)APIError{
	return LogBackendEvent(tx,Evt_complete_run,runId,nil,events)
}

func logWaitChildRun(tx*gorm.DB,runId string,events EventsTrack) APIError{
	return LogBackendEvent(tx,Evt_wait_child,runId,nil,events)
}

func logCleanRun(tx*gorm.DB,runId string, events EventsTrack) APIError {
	return LogBackendEvent(tx,Evt_clean_run,runId,nil,events)
}

func logDiscardRun(tx*gorm.DB,runId string,events EventsTrack) APIError{
	return LogBackendEvent(tx,Evt_discard_run,runId,nil,events)
}


//@todo:  lab runs cannot refer to another lab run ?
func logClearLab(tx*gorm.DB,labId uint64,events EventsTrack) APIError{
	return LogBackendEvent(tx,Evt_clear_lab,fmt.Sprintf("%d",labId),nil,events)
}
