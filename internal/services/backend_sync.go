
package services

import "github.com/apulis/bmod/ai-lab-backend/internal/models"

type BackendEventSync struct {

}

var event_sync BackendEventSync

func (d BackendEventSync)NotifyWithEvent(evt string){

}

func InitServices(){

	 models.SetEventNotifier(event_sync)
}

func QuitServices(){


}
