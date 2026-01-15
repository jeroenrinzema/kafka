# Solution: Debugging Challenge

This file contains the solutions to all issues in the debugging challenge. **Don't look until you've tried to solve them yourself!**

## Issues Found

### Producer Issues

#### Issue 1: Wrong Bootstrap Server Address
**Location:** `producer/main.go`, line 28
```go
kgo.SeedBrokers("localhost:9093"),  // Wrong port!
```

**Problem:** The broker is running on port `9092`, not `9093`.

**Fix:**
```go
kgo.SeedBrokers("localhost:9092"),
```

**Impact:** Producer cannot connect to Kafka, all messages fail to send.

---

#### Issue 2: No Acknowledgment (Fire-and-Forget)
**Location:** `producer/main.go`, line 29
```go
kgo.RequiredAcks(kgo.NoAck()),  // Risky for production!
```

**Problem:** `NoAck()` means the producer doesn't wait for broker confirmation. Messages can be lost if the broker fails.

**Fix:**
```go
kgo.RequiredAcks(kgo.AllISRAcks()),  // Wait for all replicas
// or at minimum:
kgo.RequiredAcks(kgo.LeaderAck()),   // Wait for leader
```

**Impact:** Messages may be lost without any error notification.

---

#### Issue 3: Wrong Partitioning Key
**Location:** `producer/main.go`, line 54
```go
Key:   []byte(order.OrderID),  // Should be UserID!
```

**Problem:** Using `OrderID` as the key means orders from the same user can go to different partitions. For user-specific processing, we want all orders from the same user in the same partition.

**Fix:**
```go
Key:   []byte(order.UserID),
```

**Impact:** Orders from the same user are scattered across partitions, making user-based aggregations difficult.

---

### Consumer Issues

#### Issue 4: Wrong Topic Name
**Location:** `consumer/main.go`, line 30
```go
kgo.ConsumeTopics("order-events"),  // Wrong topic name!
```

**Problem:** The producer sends to `orders`, but the consumer reads from `order-events`.

**Fix:**
```go
kgo.ConsumeTopics("orders"),
```

**Impact:** Consumer never receives any messages.

---

#### Issue 5: Reading from End of Topic
**Location:** `consumer/main.go`, line 31
```go
kgo.ConsumeResetOffset(kgo.NewOffset().AtEnd()),  // Misses historical data!
```

**Problem:** `AtEnd()` means the consumer only reads messages that arrive after it starts. All existing messages are skipped.

**Fix:**
```go
kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()),  // Read from beginning
```

**Impact:** Consumer misses all messages that were produced before it started.

---

#### Issue 6: Auto-Commit Without Consideration
**Location:** Consumer uses default auto-commit behavior

**Problem:** While auto-commit works for many use cases, it's not ideal when you need exactly-once semantics or when processing has external side effects (database writes, API calls, etc.).

**Fix:** For critical operations, use manual commits:
```go
kgo.DisableAutoCommit(),  // In client config

// After processing successfully:
client.CommitUncommittedOffsets(ctx)
```

**Impact:** Messages might be marked as consumed before processing completes, or could be reprocessed if the consumer crashes.

## Key Learnings

1. **Always verify configuration values** - broker addresses, ports, topic names
2. **Choose appropriate acknowledgment modes** - balance between performance and reliability
3. **Set realistic timeouts** - too short causes false failures, too long causes delays
4. **Partition strategically** - use meaningful keys for related data
5. **Understand offset management** - know when to read from beginning vs end
6. **Handle timeouts gracefully** - don't let consumers hang indefinitely
7. **Consider commit strategies** - auto-commit vs manual commit based on use case
8. **Test incrementally** - fix one issue at a time and verify each fix
