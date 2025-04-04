package services

import (
	"database/sql"
	"sync"

	"yourproject/models"
	"yourproject/internal/logger"
)

// DefaultJobConsumer implementa um consumer que processa os jobs.
type DefaultJobConsumer struct{}

func (c *DefaultJobConsumer) Start(jobChan <-chan *models.ScanJob, dbConn *sql.DB, gitClient GitClient, scanner Scanner, cloneMaxConc, numWorkers int) {
	cloneSem := make(chan struct{}, cloneMaxConc)
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for job := range jobChan {
				logger.Log.Debugf("[Consumer Worker %d] Processando job: %s", workerID, job.ScanID)
				if err := ProcessJob(job, dbConn, gitClient, scanner, cloneSem); err != nil {
					logger.Log.Errorf("[Consumer Worker %d] Erro no job %s: %v", workerID, job.ScanID, err)
				} else {
					logger.Log.Debugf("[Consumer Worker %d] Job %s finalizado com sucesso", workerID, job.ScanID)
				}
			}
		}(i)
	}
	wg.Wait()
}
