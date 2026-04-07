package main

// Imports
import (
    "context"
    "log"
    "net"
	"io"
	"os"
	
	"database/sql"
    _ "github.com/lib/pq"
	
	"google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
    "google.golang.org/grpc"
    pb "github.com/joshkwka/gopher-queue/api/proto/v1" 
)

// Server Struct
type server struct {
    pb.UnimplementedQueueServiceServer
	db *sql.DB
}

// Interface
func (s *server) SubmitTask(ctx context.Context, req *pb.TaskRequest) (*pb.TaskResponse, error) {
    log.Printf("Received task: %v", req.TaskType)
	// Write incoming task to DB
	insertQuery := `
		INSERT INTO tasks (task_type, payload, priority, state) 
		VALUES ($1, $2, $3, $4) 
		RETURNING id
	`

	// Variable to hold the newly generated ID
	var newId int32 

	// Execute query, pass variables, and scan returned ID into newId
	err := s.db.QueryRowContext(
		ctx, 
		insertQuery, 
		req.TaskType, 
		req.Payload, 
		req.Priority, 
		"PENDING",
	).Scan(&newId)

	// Handle database errors
	if err != nil {
		log.Printf("Database insert failed: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to save task to memory bank")
	}

	log.Printf("Task successfully saved to DB with ID: %d", newId)

	// Return task ID from DB
	return &pb.TaskResponse{TaskId: newId, Accepted: true}, nil
}

func (s *server) GetWork(ctx context.Context, req *pb.WorkRequest) (*pb.WorkResponse, error) {
    log.Printf("Worker %s requesting work", req.WorkerId)

	getWorkQuery := `
		UPDATE tasks 
		SET state = 'RUNNING' 
		WHERE id = (
			SELECT id FROM tasks 
			WHERE state = 'PENDING' 
			ORDER BY priority DESC, id ASC 
			LIMIT 1 
			FOR UPDATE SKIP LOCKED
		) 
		RETURNING id, payload;
	`

	var taskId int32
	var payload []byte

	// Execute query
	err := s.db.QueryRowContext(ctx, getWorkQuery).Scan(&taskId, &payload)

	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("No work available for Worker %s", req.WorkerId)
			// Not found error
			return nil, status.Errorf(codes.NotFound, "no pending tasks in queue")
		}
		// Catch database failures
		log.Printf("Database error fetching work: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to fetch work from memory bank")
	}

	log.Printf("Assigned Task ID %d to Worker %s", taskId, req.WorkerId)

	// Return the raw values, not pointers
	return &pb.WorkResponse{TaskId: taskId, Payload: payload}, nil
}

func (s *server) ReportStatus(stream pb.QueueService_ReportStatusServer) error {
    log.Printf("Status stream opened")

	for {
		// 1. Listen for the next heartbeat from the worker
		req, err := stream.Recv()
		if err == io.EOF {
			log.Println("Worker finished task and closed stream gracefully.")
			return nil
		}
		if err != nil {
			log.Printf("Stream received error: %v", err)
			return err
		}

		// 2. Update the Database with the current progress
		statusQuery := `UPDATE tasks SET state = $1 WHERE id = $2`
		_, dbErr := s.db.Exec(statusQuery, req.CurrState.String(), req.TaskId)
		if dbErr != nil {
			log.Printf("Failed to update task %d in DB: %v", req.TaskId, dbErr)
		}

		log.Printf("Task %d Update: %s (%.0f%%)", req.TaskId, req.CurrState.String(), req.PercComplete)

		// 3. Send Acknowledgment
		err = stream.Send(&pb.ServerSignal{Cancel: false})
		if err != nil {
			return err
		}
	}
}

func main() {
    // 1. Open the port and catch any errors
    lis, err := net.Listen("tcp", ":50051") // make configurable later
    if err != nil {
        log.Fatalf("failed to listen: %v", err)
    }

	// 2. Connect to Database
	connectionString := os.Getenv("DATABASE_URL")
	if connectionString == "" {
        connectionString = "postgres://gopher:secret@localhost:5432/gopherqueue?sslmode=disable"
        log.Println("Using local database connection string")
    } else {
        log.Println("Using database connection string from environment")
    }

	conn, err := sql.Open("postgres", connectionString)
	if err != nil {
		log.Fatalf("failed to open DB: %v", err)
	}
	defer conn.Close()

	// Test the actual connection
	if err := conn.Ping(); err != nil {
		log.Fatalf("failed to ping DB: %v", err)
	}
	log.Println("Successfully connected to PostgreSQL!")

	// 3. Ensure table exists
	createTableQuery := `
	CREATE TABLE IF NOT EXISTS tasks (
		id SERIAL PRIMARY KEY,
		task_type VARCHAR(50),
		payload BYTEA,
		priority INT,
		state VARCHAR(20)
	);`
	if _, err := conn.Exec(createTableQuery); err != nil {
		log.Fatalf("failed to create table: %v", err)
	}
	log.Println("Database schema initialized.")

    // 4. Create the gRPC engine
    grpcServer := grpc.NewServer()

    // 5. Register custom 'server' struct with the engine
    pb.RegisterQueueServiceServer(grpcServer, &server{db: conn})

    // 6. Start listening
    log.Printf("Control Plane listening at %v", lis.Addr())
    if err := grpcServer.Serve(lis); err != nil {
        log.Fatalf("failed to serve: %v", err)
    }
}