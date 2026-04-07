package main

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	pb "github.com/joshkwka/gopher-queue/api/proto/v1"
)

func main() {
	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewQueueServiceClient(conn)

	// Bump timeout to 15s to allow for the 5s simulation + network overhead
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	// 1. Open the bidirectional stream
	stream, err := client.ReportStatus(ctx)
	if err != nil {
		log.Fatalf("could not open stream: %v", err)
	}

	log.Println("Starting Task ID: 1 simulation...")

	// 2. Simulate working (5 steps, 1 second each)
	for i := 1; i <= 5; i++ {
		time.Sleep(time.Second * 1)
		
		perc := float32(i * 20)
		
		err := stream.Send(&pb.WorkerUpdate{
			TaskId:       1,
			WorkerId:     99,
			PercComplete: perc,
			CurrState:    pb.TaskState_RUNNING,
		})
		if err != nil {
			log.Fatalf("failed to send heartbeat: %v", err)
		}

		// 3. Listen for the server's acknowledgment
		signal, err := stream.Recv()
		if err != nil {
			log.Fatalf("failed to receive server signal: %v", err)
		}
		log.Printf("Progress: %.0f%% | Server Cancel Signal: %v", perc, signal.Cancel)
	}

	// 4. Final 'Completed' message
	stream.Send(&pb.WorkerUpdate{
		TaskId:       1,
		WorkerId:     99,
		PercComplete: 100,
		CurrState:    pb.TaskState_COMPLETED,
	})

	// 5. Tell the server we are done
	stream.CloseSend()
	log.Println("Task complete. Stream closed.")
}