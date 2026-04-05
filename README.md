# Gopher-Queue: Distributed Task Orchestrator

A high-performance, distributed task execution engine built to bridge low-level systems architecture with modern cloud-native orchestration. This project serves as a transition from C++/Systems to Golang/Infrastructure, implementing a scalable control plane that manages containerized worker pools.

## The Stack

* **Language:** Golang (utilizing Goroutines and Channels for high-concurrency task management).
* **Communication:** gRPC and Protocol Buffers for strictly-typed, low-latency inter-service communication.
* **Orchestration:** Kubernetes (K8s) to manage worker lifecycles, self-healing, and dynamic scaling.
* **Infrastructure:** Terraform for automated provisioning of the cluster environment.
* **Persistence:** PostgreSQL for task metadata persistence and fault-tolerant state tracking.

## Project Goals

* **Distributed Task Scheduling:** Implement a centralized Control Plane that distributes computational tasks to a pool of heartbeating Workers.
* **Cloud-Native Scalability:** Utilize a custom Kubernetes Controller to dynamically spin up or terminate Worker Pods based on real-time queue depth.
* **Fault Tolerance:** Achieve "at-least-once" delivery by implementing worker health monitoring; if a heartbeat fails, tasks are automatically re-queued.
* **Performance Optimization:** Leverage Go's concurrency primitives to handle thousands of simultaneous task updates with minimal overhead.

## Architecture

* **Control Plane:** A gRPC server that acts as the "Brain," maintaining a thread-safe state machine of all tasks in the system.
* **Workers:** Stateless, containerized Go binaries that pull tasks, execute workload logic, and report status via gRPC streams.
* **Persistence Layer:** A normalized relational schema in PostgreSQL to ensure task data survives Control Plane restarts.

## Roadmap

* [ ] **Phase 1:** Define gRPC Service and implement basic Go Client/Server communication.
* [ ] **Phase 2:** Integrate PostgreSQL for persistent task state and history.
* [ ] **Phase 3:** Containerize components and deploy to a local Minikube cluster.
* [ ] **Phase 4:** Implement Horizontal Pod Autoscaling (HPA) and Terraform scripts for cloud deployment.
