
package services

import (
	"github.com/apulis/bmod/ai-lab-backend/internal/loggers"
	"github.com/apulis/bmod/ai-lab-backend/internal/models"
	"github.com/apulis/bmod/ai-lab-backend/pkg/exports"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

const (
	read_db_failed_retry    = 5000
	default_task_fail_delay = 5000
	default_continue_do     = 20
)

type BackendEventSync struct {

	 max_event_id  uint64
	 sync_group    sync.WaitGroup

	 tasks         map[string]*BackendTask
}


var g_backend_srv BackendEventSync

func (d *BackendEventSync)NotifyWithEvent(evt string,lastId uint64){

	d.SetLastEventID(lastId)
    if task,ok := d.tasks[evt];ok {
        task.Notify()
	}else{//should never happen
		log.Fatalf("no handler for event name:%s",evt)
	}
}
func (d*BackendEventSync)JobStatusChange(runId string){
	//@todo: notify status change to application ???

}

func (d*BackendEventSync)SetLastEventID(lastId uint64){
	for{//keep last processed id increased
		v := atomic.LoadUint64(&d.max_event_id)
		if v>= lastId {
			break
		}else if atomic.CompareAndSwapUint64(&d.max_event_id,v,lastId){
			break
		}
	}
}

func (d*BackendEventSync)AddTaskQueue(name string, process func(event *models.Event)APIError ,fail_delay int,
	        run func(*BackendTask) ) {
	  if d.tasks == nil{
	  	d.tasks=make(map[string]*BackendTask)
	  }
	  if _,ok := d.tasks[name];ok{
		  panic("backend task name exists!!!")
	  }
	  if fail_delay == 0 {
	  	fail_delay = default_task_fail_delay
	  }
	  if process == nil {
	  	panic("processor cannot be nil")
	  }
	  if run == nil {
	  	run = default_task_run
	  }
	  task := &BackendTask{
		  name:           name,
		  notify:         make(chan int,1),
		  failed_retry:   fail_delay,
		  task_run:       run,
		  processor:      process,
	  }
	  d.tasks[name]=task

	  d.sync_group.Add(1)
	  go run(task)
}

func  (d*BackendEventSync)QuitAllTask() {
	  for _,item := range(d.tasks) {
	  	 item.Quit()
	  }
	  d.sync_group.Wait()
	  log.Printf("all backend tasks quit !")
}

func (d*BackendEventSync)Done(){
	  d.sync_group.Done()
}

func (d*BackendEventSync) SyncMaxBackendEvent()APIError{
	 if max,err := models.GetMaxBackendEventID();err != nil{
	 	return err
	 }else{
		 d.SetLastEventID(max)
		 return nil
	 }
}
func (d*BackendEventSync) GetMaxBackendEvent() uint64 {
	 return atomic.LoadUint64(&d.max_event_id)
}


type BackendTask struct{
	 // task name
	 name           string
	 // last porcessed event id
	 last_processed uint64
	 // push 0 means continue event ; otherwise should quit go routine
	 notify chan    int
	 // process failed retry delay
	 failed_retry   int
	 // task run
	 task_run       func( *BackendTask)
	 // tick processed event
	 processor      func(*models.Event) APIError
}
// non-block send 1 , safe when channel closed
func (d*BackendTask) Notify(){
	defer func() { recover() } ()
	select{
	   // panic if ch is closed
	   case d.notify<- 1 :
	   default:
	}
}
// close channel, then always return 0 when wait
func (d*BackendTask) Quit (){
	close(d.notify)
}
func (d*BackendTask) Wait() bool {
	v := <- d.notify
	return v == 1
}
func (d*BackendTask) AlertableWait(ts int , format string , v...interface{}) {
	logger.Errorf(format,v...)
	timer := time.NewTimer(time.Duration(ts) * time.Millisecond)
	defer timer.Stop()

	for {// wait until timeout or closed
		select {
		 case v := <- d.notify:
		 	if v == 0 {
		 		return
			}else{
				continue
			}
		 case <- timer.C:
		 	    return
		}
	}
}
func (d*BackendTask)SetLastProcessed(id uint64) {
	 d.last_processed=id
}

func getBackendSyncer() *BackendEventSync{
	return &g_backend_srv
}

func InitServices() error {
	 logger = loggers.GetLogger()
	 models.SetEventNotifier(getBackendSyncer())
	 if  err := getBackendSyncer().SyncMaxBackendEvent();err != nil {
		 logger.Fatalf("sync max backend event id failed:%s",err.Error())
	 	return err
	 }
	 if rolls,err := models.RollBackAllPrepareFailed();err != nil {
	 	logger.Fatalf("rollback all prepare failed runs error:%s",err.Error())
	 	return err
	 }else if rolls > 0{
	 	logger.Warnf("rollback all prepare failed runs num :%d",rolls)
	 }
	 getBackendSyncer().AddTaskQueue(models.Evt_init_run, InitProcessor,0,nil)
	 getBackendSyncer().AddTaskQueue(models.Evt_start_run,StartProcessor,0,nil)
	 getBackendSyncer().AddTaskQueue(models.Evt_kill_run, KillProcessor,0,nil)
	 getBackendSyncer().AddTaskQueue(models.Evt_complete_run, SaveProcessor,0,nil)
	 getBackendSyncer().AddTaskQueue(models.Evt_clean_run,CleanProcessor,0,nil)
	 getBackendSyncer().AddTaskQueue(models.Evt_discard_run,DiscardProcessor,0,nil)
	 getBackendSyncer().AddTaskQueue(models.Evt_clear_lab,ClearLabProcessor,0,nil)
	 return nil
}

func QuitServices(){
     getBackendSyncer().QuitAllTask()
}

func default_task_run(task*BackendTask){

		var event models.Event
		var err   APIError

		defer getBackendSyncer().Done()

		for {
			if !do_read_event(task,&event) {
				break
			}

			err = task.processor(&event)
			if err == nil {
				err = models.RemoveBackEvent(event.ID)
			}
			if err == nil {
                task.SetLastProcessed(event.ID)
			}else {
				ts := task.failed_retry
				if err.Errno() == exports.AILAB_WOULD_BLOCK {
					ts = default_continue_do
				}
				task.AlertableWait(ts,"process backend event type:%s id:%d failed:%s",
					  event.Type,event.ID, err.Error())
			}
		}
}


//@mark: will quit when read none-event
func do_read_event(task*BackendTask,event*models.Event) bool {
	event.ID=0
	// always non-block when go here
	task.Notify()
	for {
		if !task.Wait() {
			return false
		}
		max := getBackendSyncer().GetMaxBackendEvent()
		err := models.ReadBackendEvent(event,task.name,task.last_processed)
		if err == nil {// wait for next event retry
			task.Notify()
			return true
		}else if err.Errno() == exports.AILAB_NOT_FOUND {//block until notify by db thread
			task.SetLastProcessed(max)
			continue
		}else{
            task.AlertableWait(read_db_failed_retry,"read back event log failed:%s",err.Error())
            task.Notify()
            continue
		}
	}
}
