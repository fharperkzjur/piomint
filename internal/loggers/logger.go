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
package loggers

import (
	"fmt"
	loggerconfs "github.com/apulis/simple-gin-logger/pkg/configs"
	loggergen "github.com/apulis/simple-gin-logger/pkg/loggers"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"math"
	"time"
)

var logger *logrus.Logger

func InitLogger(conf *loggerconfs.LogConfig) (*logrus.Logger, error) {
	var err error
	logger, err = loggergen.InitLogger(conf)
	return logger, err
}

func GetLogger() *logrus.Logger {
	return logger
}

func GinLogger(logger logrus.FieldLogger) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		start := time.Now()
		c.Next()
		stop := time.Since(start)
		latency := int(math.Ceil(float64(stop.Nanoseconds()) / 1000.0))
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()
		clientUserAgent := c.Request.UserAgent()
		referer := c.Request.Referer()
		dataLength := c.Writer.Size()
		if dataLength < 0 {
			dataLength = 0
		}

		entry := logger.WithFields(logrus.Fields{
			"clientIP":   clientIP,
			"referer":    referer,
			"dataLength": dataLength,
			"userAgent":  clientUserAgent,
		})

		if len(c.Errors) > 0 {
			entry.Error(c.Errors.ByType(gin.ErrorTypePrivate).String())
		} else {
			msg := fmt.Sprintf("%s: %s <%d> (%dμs)", c.Request.Method, path, statusCode, latency)
			if statusCode > 499 {
				entry.Error(msg)
			} else if statusCode > 399 {
				entry.Warn(msg)
			} else {
				entry.Info(msg)
			}
		}
	}
}
