/* ******************************************************************************
* 2019 - present Contributed by Apulis Technology (Shenzhen) Co. LTD
*
* This program and the accompanying materials are made available under the
* terms of the MIT License, which is available at
* https://www.opensource.org/licenses/MIT
*
* See the NOTICE file distributed with this work for additional
* information regarding copyright ownership.
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
* WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
* License for the specific language governing permissions and limitations
* under the License.
*
* SPDX-License-Identifier: MIT
******************************************************************************/
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

	notify chan  exports.NotifierData

}

func (d*AsyncWSNotifier) Notify(notify*exports.NotifierData){
	d.notify <- *notify
}

func (d*AsyncWSNotifier) Fetch() exports.NotifierData{
	notify := <- d.notify
	return notify
}

func (d*AsyncWSNotifier) Quit() {
	close(d.notify)
	d.notify=nil
}

func (d*AsyncWSNotifier) Run() {

	 for {
	 	notify := d.Fetch()
	 	if len(notify.Cmd) == 0 {
	 		break
	    }
	    scope,_   := notify.Scope.(*models.RunNotifyScope)
	    payload,_ := notify.Payload.(*models.RunNotifyPayload)
	    if notify.Scope != nil{
		     notifyRunStatusChange(scope,payload,notify.Cmd)
		     if notify.Cmd == exports.AILAB_NOTIFY_CMD_DEL_RUN {
		     	runNotifierData := &models.RunStatusNotifier{
			        RunNotifyScope:   *scope,
		        }
		        if payload != nil{
		        	runNotifierData.RunNotifyPayload=*payload
		        }
		     	postMessageCenterRunMsg(runNotifierData,exports.AILAB_MCT_MESSAGE_TYPE_DEL)
		     }
	    }
	 }
}

func notifyRunCreated(notifier * models.RunStatusNotifier) APIError {
  	  postMessageCenterRunMsg(notifier,exports.AILAB_MCT_MESSAGE_TYPE_NEW)
	  return notifyRunStatusChange(&notifier.RunNotifyScope, &notifier.RunNotifyPayload,exports.AILAB_NOTIFY_CMD_NEW_RUN)
}

func notifyRunComplete(notifier * models.RunStatusNotifier) APIError {
  	  postMessageCenterRunMsg(notifier,exports.AILAB_MCT_MESSAGE_TYPE_COMPLETE)
	  return notifyRunStatusChange(&notifier.RunNotifyScope, &notifier.RunNotifyPayload,exports.AILAB_NOTIFY_CMD_STATUS_RUN)
}

func notifyRunStatusChange(scope*models.RunNotifyScope,payload*models.RunNotifyPayload,cmd string) APIError{
	 notifyData := wsmsg.PushMsg{
		 Header:  wsmsg.PushMsgHeader{
			 ReqId:       utils.GenerateReqId(),
			 ModId:       exports.AILAB_MODULE_ID,
			 Cmd:         cmd,
			 PushType:    wsmsg.PushTypeUserGroup,
			 PushTargets: []string{strconv.FormatUint(scope.UserGroupId,10)},
		 },
		 Payload: scope,
		 Scope:   payload,
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
	 }else{
	 	logger.Infof("postMessageCenterRunMsg with topic %s message:%v success !",wsmsg.MESSAGE_CENTER_TOPIC_NAME,msg)
	 }
	 return err
}




