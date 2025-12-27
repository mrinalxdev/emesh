# The problem

In distributed systems, when multiple nodes concurrently update shared state (like a KV store), inconsistent views can arise if messages are delivered in different orders across nodes. This leads 

- Node A sees "comment → like", but Node B sees "like → comment", breaking logical cause-and-effect.

- A get might return outdated or missing values because updates weren’t properly synchronized.

- Without observability, it’s impossible to tell if messages are stuck, reordered, or dropped.

Traditional approaches like strong consensus (e.g., Raft) are overkill for many edge or IoT scenarios where availability, low latency, and partial connectivity matter more than linearizability.

# How this prototype tends to solves it 

I am trying to implement casual broadcasting protocol using vector clocks which states 

> If event A causally precedes event B, then every node delivers A before B.

#### How is this working 

1. Vector Clocks for Causality Tracking
  - Each node maintains a vector clock (e.g., {n1:2, n2:1}).
  - On put, the sender increments its own clock and attaches the full vector to the message.
  - Receivers buffer messages until all prior causal dependencies are satisfied (i.e., no gaps in the vector).

2. UDP-Based, Lightweight Transport
  - Uses raw UDP for efficiency (suitable for high-throughput or loss-tolerant edge environments).
  - No TCP overhead, yet causality is preserved even over unreliable networks.

3. Usage of Prometheus
  - Exposes metrics like 
    -  emesh_reorder_depth: How many messages were buffered before delivery
    - emesh_deliver_lag_ms: Delivery latency due to causal waiting
    - emesh_recv_bytes_total: Traffic volume per node
  
  - Enables operators to detect bottlenecks, partition effects, or stuck messages.