package main

import (
	"encoding/json"
	"fmt"
)

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

func main(){


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