package main

import (
	"context"
	"log"
	"time"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	pb "github.com/joshkwka/gopher-queue/api/proto/v1"
)

func main() {
	serverAddr := os.Getenv("CONTROL_PLANE_URL")
	if serverAddr == "" {
		serverAddr = "localhost:50051"
	}
	
	conn, err := grpc.NewClient(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewQueueServiceClient(conn)

	log.Println("Worker active. Waiting for tasks...")

	for {
		// Create fresh context for each task
		pollCtx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		res, err := client.GetWork(pollCtx, &pb.WorkRequest{WorkerId: 99})
		cancel() 

		if err != nil {
			log.Printf("No work found: %v. Retrying in 5s...", err)
			time.Sleep(time.Second * 5)
			continue 
		}

		log.Printf("Fetched Task ID: %d. Starting processing...", res.TaskId)


		workCtx, workCancel := context.WithTimeout(context.Background(), time.Minute*1)
		stream, err := client.ReportStatus(workCtx)
		if err != nil {
			log.Printf("Failed to open stream: %v", err)
			workCancel()
			continue
		}

		// Simulation Loop
		for i := 1; i <= 5; i++ {
			time.Sleep(time.Second * 1)
			perc := float32(i * 20)
			
			err := stream.Send(&pb.WorkerUpdate{
				TaskId:       res.TaskId, 
				WorkerId:     99,
				PercComplete: perc,
				CurrState:    pb.TaskState_RUNNING,
			})
			if err != nil {
				log.Printf("Heartbeat failed: %v", err)
				break
			}

			_, err = stream.Recv()
			if err != nil {
				log.Printf("Failed to receive ack: %v", err)
				break
			}
			log.Printf("Task %d Progress: %.0f%%", res.TaskId, perc)
		}

		stream.CloseSend()
		workCancel()
		log.Println("Task finished. Polling for next...")
	}
}