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
package main

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"github.com/apulis/bmod/ai-lab-backend/internal/models"
	"github.com/apulis/bmod/ai-lab-backend/internal/services"
	"gorm.io/gorm"
	"log"
	"strconv"
)

var db * gorm.DB

type LabRunStats struct{
	RunStarting  int  `json:"start"`
	Running      int  `json:"run"`
	Stopping     int  `json:"kill"`
	Fails        int  `json:"fail"`
	Errors       int  `json:"error"`
	Aborts       int  `json:"abort"`
	Success      int  `json:"success"`
}

type JobStats  map[string]*LabRunStats

func(d*JobStats)StatusChange(jobType string,from,to int){
	var jobs * LabRunStats
	if stats,ok := (*d)[jobType];!ok {
		jobs = &LabRunStats{}
		(*d)[jobType]=jobs
	}else{
		jobs = stats
	}
	jobs.Aborts++
	jobs.Fails--
}

func init_db() (err error){
	db,err = models.OpenDB("postgres","127.0.0.1",5432,"postgres","root","test_river")
	db=db.Debug()
	return
}

type TestNull struct{
	gorm.Model
	Time        int
	Description string
	Hello       SQLInt
	Value       SQLString
}

type SQLInt     int
type SQLString  string

func (s*SQLString) Scan(value interface{}) error{
    switch v := value.(type) {
      case string: *s = SQLString(v)
      case int64:  *s = SQLString(strconv.FormatInt(v,10))
    }
    return nil
}
func (s *SQLString) Value() (driver.Value,error){
	 return *s,nil
}


// Scan implements the Scanner interface.
func (n *SQLInt) Scan(value interface{}) error {
	if value == nil {
		*n = 0
		return nil
	}
	switch  v := value.(type) {
	  case int64: *n =  SQLInt(v)
	  case string: i,_ := strconv.ParseInt(v,0,32)
	                 *n=SQLInt(i)
	  default:  *n=0
	}
	return nil
}

// Value implements the driver Valuer interface.
func (n SQLInt) Value() (driver.Value, error) {
	return int64(n),nil
}

func test_read_db(){

	data :=  TestNull{}

	var ival1,ival2 SQLInt
	var sval1       SQLString

	err := db.Model(&data).Where("hello is null").Select("time,hello,value").Row().Scan(&ival1,&ival2,&sval1)

    var aaa string = string(sval1)


	log.Printf("read data from db:%+v",err)

}

func test_db(){

	 db.AutoMigrate(&TestNull{})
	 
	 db.Create(&TestNull{
		 Model:       gorm.Model{},

	 })


}



func main(){

	 init_db()

	test_read_db()

	 cmds,mounts := services.CheckResourceMounts([]string{
      "iqi-laucnher","{{model/code}}",
	 }, map[string]interface{}{
	 	"model":map[string]interface{}{
	 		"path":"pvc://abcd/1234",
	 		"subResource":map[string]interface{}{
	 			"dataset":"pvc://dddd",
			},
		},

	 })

	 fmt.Printf("cmds:%+v mounts:%+v",cmds,mounts)

	 var sss interface{}

	 abc := make(map[string]int,10)

	 fmt.Printf("map:%v",abc)

	 sss = map[string]int{}

	 vv ,_:= sss.(map[string]int)

	 fmt.Println(vv)

	 stats := &JobStats{}

	 stats.StatusChange("train",1,3)
	 stats.StatusChange("eval",1,3)
	 stats.StatusChange("train",2,3)
	 stats.StatusChange("eval",1,3)

	 str,_ := json.Marshal(stats)

	 fmt.Println(string(str))


	 var v = make(map[string]interface{})

	 v["123"]=map[string]int {}

	 fmt.Println("aa",v["123"])

}