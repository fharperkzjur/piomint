
package services

import (
	"fmt"
	"github.com/apulis/bmod/ai-lab-backend/internal/models"
	"github.com/apulis/bmod/ai-lab-backend/internal/utils"
	"github.com/apulis/bmod/ai-lab-backend/pkg/exports"

	"github.com/apulis/go-business/pkg/wsmsg"
	"strconv"
)

type AsyncWSNotifier struct{

	notify chan  string

}

func (d*AsyncWSNotifier) Notify(runId string){
	d.notify <- runId
}

func (d*AsyncWSNotifier) Fetch() string{
	runId := <- d.notify
	return runId
}

func (d*AsyncWSNotifier) Quit() {
	close(d.notify)
	d.notify=nil
}

func (d*AsyncWSNotifier) Run() {

	 for {
	 	runId := d.Fetch()
	 	if len(runId) == 0 {
	 		break
	    }
		if notifyData,err := models.QueryRunNotifierData(runId);err == nil{
				if notifyData.DeletedAt > 0 {// run deleted
                   notifyRunDeleted(notifyData)
				}else if notifyData.Status == exports.AILAB_RUN_STATUS_INIT {// new run
                   notifyRunCreated(notifyData)
				}else if exports.IsRunStatusNonActive(notifyData.Status) {// run complete
                   notifyRunComplete(notifyData)
				}else{// run status change
				   notifyRunStatusChange(notifyData,exports.AILAB_NOTIFY_CMD_STATUS_RUN)
				}
		}else if err.Errno() == exports.AILAB_NOT_FOUND{//should not error
			logger.Warnf("query runId %s for async notify not found !",runId)
		}else{
			logger.Errorf("query runId %s for async notify error:%s drop it !",runId,err.Error())
		}
	 }
}

func notifyRunDeleted(notifier * models.RunStatusNotifier) APIError {
      postMessageCenterRunMsg(notifier,exports.AILAB_MCT_MESSAGE_TYPE_DEL)
	  return notifyRunStatusChange(notifier,exports.AILAB_NOTIFY_CMD_DEL_RUN)
}

func notifyRunCreated(notifier * models.RunStatusNotifier) APIError {
  	  postMessageCenterRunMsg(notifier,exports.AILAB_MCT_MESSAGE_TYPE_NEW)
	  return notifyRunStatusChange(notifier,exports.AILAB_NOTIFY_CMD_NEW_RUN)
}

func notifyRunComplete(notifier * models.RunStatusNotifier) APIError {
  	  postMessageCenterRunMsg(notifier,exports.AILAB_MCT_MESSAGE_TYPE_COMPLETE)
	  return notifyRunStatusChange(notifier,exports.AILAB_NOTIFY_CMD_STATUS_RUN)
}

func notifyRunStatusChange(notifier * models.RunStatusNotifier,cmd string) APIError{
	 notifyData := wsmsg.PushMsg{
		 Header:  wsmsg.PushMsgHeader{
			 ReqId:       utils.GenerateReqId(),
			 ModId:       exports.AILAB_MODULE_ID,
			 Cmd:         cmd,
			 PushType:    wsmsg.PushTypeUserGroup,
			 PushTargets: []string{strconv.FormatUint(notifier.UserGroupId,10)},
		 },
		 Payload: &notifier.RunNotifyPayload,
		 Scope:   &notifier.RunNotifyScope,
	 }
	 url := fmt.Sprintf("%s/ws/inner/publish")
	 return DoRequest(url,"PUT",nil,&notifyData,nil)
}

func postMessageCenterRunMsg(notifier*models.RunStatusNotifier,MsgType string) APIError{
	 msg := notifier.GetPublishMsg(MsgType)
	 if msg == nil {//skip store this message
	 	return nil
	 }
	 err := publishMsg(wsmsg.MESSAGE_CENTER_TOPIC_NAME,msg)
	 if err != nil {
	 	logger.Warnf("postMessageCenterRunMsg with message:%v failed:%s",msg,err.Error())
	 }
	 return err
}




