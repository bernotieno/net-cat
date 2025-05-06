package broadcast

import (
	"netcat/models"
	"netcat/utils"
)

func Broadcaster() {
	for msg := range models.Broadcast {
		utils.LogToFile(msg)

		models.Mu.Lock()
		for conn := range models.Clients {
			_, err := conn.Write([]byte(msg))
			if err != nil {
				conn.Close()
				delete(models.Clients, conn)
			}
		}
		models.Mu.Unlock()
	}
}
