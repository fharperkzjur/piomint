
package services


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
	    //@todo:
	    logger.Infof("runId %s status changed ")
	 }
}







