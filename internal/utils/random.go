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
package utils

import (
	"math/rand"
	"strconv"
	"time"
)

var letterRunes = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

var r* rand.Rand

var nletter = len(letterRunes)

func init() {
	r = rand.New(rand.NewSource(time.Now().UnixNano()))
}

func GenerateRandomStr(n int) string {
	b := make([]byte, n)

	for i := range b {
		b[i] = letterRunes[r.Intn(nletter)]
	}
	return string(b)
}

func GenerateRandomPasswd(n int) []byte {
	b := make([]byte, n)

	for i := range b {
		b[i] = letterRunes[r.Intn(nletter)]
	}
	return b
}

func GenerateReqId() string{
	seqno := time.Now().UTC().UnixNano()/1000
	seqno = seqno*1000 + rand.Int63n(1000)
	return strconv.FormatInt(seqno,10)
}



