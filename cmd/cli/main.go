package main

import (
	"context"
	"flag"
	"log"
	"time"

	pb "github.com/joshkwka/gopher-queue/api/proto/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	// 1. Define command-line flags (Variables, Default Value, Description)
	taskType := flag.String("type", "Simulation", "The type of task to process")
	payload := flag.String("payload", "default-sensor-data", "The data the worker needs")
	priority := flag.Int("priority", 1, "Task priority (higher number = higher priority)")
	serverAddr := flag.String("server", "localhost:50051", "The Control Plane address")

	// Parse the flags from the terminal prompt
	flag.Parse()

	log.Printf("Dialing Control Plane at %s...", *serverAddr)

	// 2. Connect to the Control Plane
	conn, err := grpc.NewClient(*serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Fatal system error - could not connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewQueueServiceClient(conn)

	// 3. Set a strict timeout for the submission
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	// 4. Construct the Protobuf Request
	req := &pb.TaskRequest{
		TaskType: *taskType,
		Payload:  []byte(*payload),
		Priority: int32(*priority),
	}

	log.Printf("Submitting [%s] task with priority %d...", *taskType, *priority)

	// 5. Fire the RPC
	res, err := client.SubmitTask(ctx, req)
	if err != nil {
		log.Fatalf("Task submission rejected by cluster: %v", err)
	}

	// 6. Print the receipt!
	log.Printf("Task accepted. Assigned ID: %d", res.TaskId)
}