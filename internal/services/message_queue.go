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
	"encoding/json"
	"fmt"
	"github.com/apulis/bmod/ai-lab-backend/internal/configs"
	"github.com/apulis/bmod/ai-lab-backend/pkg/exports"
	"github.com/apulis/sdk/go-utils/broker"
	"github.com/apulis/sdk/go-utils/broker/rabbitmq"
)

var mq_broker broker.Broker

func startMQConnector() error{
	config := &configs.GetAppConfig().Rabbitmq

	if config.Port == 0 {//do nothing when disable mq
		return nil
	}

	addr := fmt.Sprintf("amqp://%s:%s@%s:%d",config.User,config.Password,config.Host,config.Port)
	mq_broker = rabbitmq.NewBroker(
		broker.Addrs(addr),
		rabbitmq.ExchangeName("default"),
	)
	if err := mq_broker.Connect(); err != nil {
		logger.Fatalf("connect rabbitmq:%s error:%s !",addr,err.Error())
		return err
	}

	//@mark: listen job monitor
	if _, err := mq_broker.Subscribe(fmt.Sprintf("%v",exports.AILAB_MODULE_ID),
		 MonitorJobStatus, rabbitmq.DurableQueue(),rabbitmq.AckOnSuccess());err != nil {
			logger.Fatalf("Subscribe rabbitmq:%s error:%s !",addr,err.Error())
			return err
	}
	return nil
}

func stopMQConnector() {
	if mq_broker != nil {
		mq_broker.Disconnect()
	}
}

func publishMsg(topic string,data interface{}) APIError{
	if mq_broker == nil {
		return exports.RaiseAPIError(exports.AILAB_MQ_SEND_ERROR,"mq broker is disabled !")
	}

	bytes,_ := json.Marshal(data)

	err := mq_broker.Publish(topic,&broker.Message{
		Header: make(map[string]string),
		Body:   bytes,
	})
	if err != nil {
		return exports.RaiseAPIError(exports.AILAB_MQ_SEND_ERROR,"mq broker publish error:" + err.Error())
	}
	return nil
}
