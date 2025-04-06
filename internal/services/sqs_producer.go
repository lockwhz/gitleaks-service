package services

import (
	"context"
	"encoding/json"
	"time"

	"yourproject/models"
	"yourproject/internal/logger"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

type SQSProducer interface {
	Start() <-chan *models.ScanJob
}

type DefaultSQSProducer struct {
	Client   *sqs.Client
	QueueURL string
}

func (p *DefaultSQSProducer) Start() <-chan *models.ScanJob {
	jobChan := make(chan *models.ScanJob, 10)
	go func() {
		ctx := context.Background()
		for {
			resp, err := p.Client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
				QueueUrl:            &p.QueueURL,
				MaxNumberOfMessages: 1,
				WaitTimeSeconds:     10,
			})
			if err != nil {
				logger.Log.Errorf("Erro ao receber mensagem da SQS: %v", err)
				time.Sleep(5 * time.Second)
				continue
			}
			if len(resp.Messages) == 0 {
				continue
			}
			msg := resp.Messages[0]
			var job models.ScanJob
			if err := json.Unmarshal([]byte(*msg.Body), &job); err != nil {
				logger.Log.Errorf("Erro ao parsear JSON da mensagem: %v", err)
				deleteMessage(ctx, p.Client, p.QueueURL, msg.ReceiptHandle)
				continue
			}
			deleteMessage(ctx, p.Client, p.QueueURL, msg.ReceiptHandle)
			logger.Log.Debugf("Producer: job %s para o repositório %s recebido", job.ScanID, job.RepositoryFullName)
			jobChan <- &job
		}
	}()
	return jobChan
}

func deleteMessage(ctx context.Context, client *sqs.Client, queueURL string, receiptHandle *string) {
	_, err := client.DeleteMessage(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      &queueURL,
		ReceiptHandle: receiptHandle,
	})
	if err != nil {
		logger.Log.Errorf("Erro ao deletar mensagem da SQS: %v", err)
	}
}
