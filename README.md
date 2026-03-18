# Elevator Project
## Project Description
The goal of this project is to design and implement a distributed control system for three elevators. Each elevator has its own local control interface, including cab call buttons, a stop button, and an obstruction switch. Hall calls, however, are shared across the system: they are visible on all elevators and must be coordinated globally. 

The system shall ensure that cab calls are handled locally by the elevator in which they are registered, while hall calls are allocated to the most suitable elevator. In addition, the system must be fault tolerant. It should continue to operate correctly under packet loss, network disconnections, motor stoppage, and software crashes. The elevators must cooperate to maintain a consistent view of orders and elevator states, so that all requests are eventually served even when failures occur. 

## Design Description
The system is based on a peer-to-peer architecture without a central coordinator. Each elevator runs the same software and periodically broadcasts its local elevator state over the network using UDP. Received states are stored in a local world state, which contains the most recent known state of all peers as well as the globally known order information. 

Hall calls are distributed using a hall request assigner that runs locally on each elevator. The assigner takes the confirmed hall requests and the current elevator states as input, and deterministically computes an assignment. Because all elevators operate on the same information, they independently arrive at the same assignment and each elevator selects the orders assigned to itself. Cab calls are not distributed, they remain local to the elevator where they were registered. 

To detect disconnected peers, the system uses periodic heartbeats together with a supervisor mechanism. If an elevator stops receiving heartbeats from another peer, that peer is marked as suspected dead. A peer is only considered dead once the remaining elevators reach consensus, reducing the risk of incorrect failure detection caused by packet loss. 

Orders are propagated using a cyclic state machine with the phases NoOrder, Unconfirmed, Confirmed, and Served. When a button is pressed, the corresponding order is first marked as unconfirmed. An order is only set to confirmed once all currently alive elevators have seen and acknowledged the unconfirmed state. Similarly, an order is only removed from the world state once all alive elevators have observed the served state. This mechanism ensures that all peers converge toward the same order view, even in the presence of unreliable communication. 

Both elevator states and order messages are broadcasted continuously. This allows a rejoining elevator to quickly reconstruct its local world state after a temporary network partition or restart, without requiring a dedicated resynchronization request. Together, the world state representation, the consensus-based peer supervision, and the cyclic order propagation mechanism provide a fault-tolerant distributed elevator system. 

## Module Overview
- Elevio: The interface to the elevator hardware. Handles sensor input, controls motor direction and lamps.  
- FSM: Implements the local elevator logic. Receives orders to do from Ordermanagement and decides movement, stopping decisions, and door behavior. 
- Ordermanagement: Uses peer liveness and broadcasted states to determine when orders are confirmed, and which orders that should be served locally. 
- Supervisor: Detects changes in peer liveness by using periodic heartbeats. The supervisor notifies the ordermanagement in case of change in liveness
- Network: Handles communication between elevators using UDP broadcasting. Transmits and receives elevator states, order messages, and heartbeats 

## Program Usage
The program is started from the command line:  

>go run . -id <elevator_id> -addr < ip:port>. 

### Example:  
>go run . -id elev1 -addr localhost:15657  
>go run . -id elev2 -addr localhost:15658  
>go run . -id elev3 -addr localhost:15659  


Each elevator instance must run with a unique ID and network address. 