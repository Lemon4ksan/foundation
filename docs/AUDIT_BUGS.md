# Foundation Bug Bounty & Audit Log

This file contains automated findings from the AI Audit Swarm.
Any inconsistencies, vulnerabilities, or thread-safety issues discovered during the documentation process are appended here.

## Discovered Issues

## Spinlock backoff exhaustion
**File:** `D:\CodingProjects\foundation\sync\spinlock\spinlock.go`
**Description:** In `lockSlow()`, the `backoff` variable is not reset if the `CompareAndSwapUint32` call fails. If a goroutine reaches the maximum backoff (32) and starts using `runtime.Gosched()`, but fails to acquire the lock via CAS, the outer loop repeats. Since `backoff` remains >= 32, subsequent inner loop executions will skip hardware PAUSE and immediately drop into OS yielding, destroying the adaptive spinning behavior for the remainder of the lock acquisition attempt.
## limiter.go Audit Report

### Vulnerabilities & Inconsistencies
1. **Zero Value Panics & Deadlocks (Inconsistency):** 
   - The documentation for `Adaptive`, `Keyed`, and `SlidingWindow` incorrectly states that "The zero value is empty and ready to use."
   - A zero value `Adaptive` has a limit of `0`, causing `Acquire` to block indefinitely.
   - A zero value `Keyed` has a nil `limiters` map, which causes a panic upon assignment in `getLimiter`.
   - A zero value `SlidingWindow` has a limit of `0`, effectively denying all requests instead of acting as a usable limiter.
2. **Compilation Error / Non-existent Method (Bug):** 
   - In `NewKeyed`, the background sweep goroutine is started with `kl.wg.Go(kl.sweepLoop)`. However, `kl.wg` is of type `sync.WaitGroup`, which does not possess a `Go` method in the standard library.
   - Furthermore, `kl.wg.Add(1)` and `kl.wg.Done()` are never called, making the `Wait()` call in `Close()` ineffective even if `Go` were available.
- **Bug in `structures/linkedlist/linkedlist.go`**: Reusing element slots in a chunk-based linked list causes `*Element[T]` pointers to alias new elements after removal. If a user retains a pointer to an element, calls `Remove(e)`, and subsequently inserts a new element, the old pointer `e` will magically reference the newly inserted element (since the same slot from the free list is reused). Calling `Remove(e)` again will erroneously remove the new element, violating standard linked-list identity semantics.

## Circuit Breaker O(N) Time Complexity Inconsistency
**File:** `D:\CodingProjects\foundation\sync\breaker\breaker.go`
**Description:** The documentation for `Breaker.Do` claims that it executes in O(1) time (excluding the execution of `fn`). However, the internal method `recordResult` called by `Do` iterates over the entire `cb.records` slice to calculate the failure ratio (`failures++` in a loop). Similarly, `pruneRecords` iterates to find expired records. Thus, `Do` executes in O(N) time, where N is the number of requests tracked within the sliding window, directly contradicting the documented O(1) time complexity.
## breaker.go Audit Report

### Vulnerabilities & Inconsistencies
1. **Zero Value Breaker Tripping (Inconsistency):** 
   - The documentation for `Breaker` claims that "The zero value for a Breaker is a valid, closed circuit breaker ready for use."
   - However, a zero-valued `Breaker` possesses a zero-valued `Config` because `ResolveDefaults` is only invoked inside `New()`.
   - With a `FailureThreshold` of `0.0` and `MinRequests` of `0`, the breaker immediately triggers `ratio >= cb.cfg.FailureThreshold` (e.g., `0.0 >= 0.0`) upon the completion of the very first request, regardless of whether it succeeds or fails.
   - This causes the zero-valued breaker to instantaneously transition to `StateOpen` on its first invocation, refuting the claim that it is "ready for use" as a closed circuit breaker.
## backoff.go Audit Report

### Vulnerabilities & Inconsistencies
1. **Zero Value Backoff Bypasses Intended Defaults (Inconsistency):** 
   - The documentation for `Exponential`, `Linear`, and `Constant` states that "The zero value is empty and ready to use."
   - However, a zero-valued instance possesses zero limits (`Initial = 0`, `Max = 0`, `Delay = 0`, `Step = 0`), which completely bypasses the safe fallbacks enforced by their respective constructors (e.g., `NewExponential` enforces an initial delay of 100ms and a max of 30s).
   - As a result, utilizing the zero value generates a backoff duration of `0` constantly, effectively resulting in a tight retry loop with no actual backoff. This contradicts the fundamental purpose of the package and the implication that the zero value is "ready to use" as a proper backoff strategy.
- **Bug in `sync/breaker/breaker.go`**: Circuit breaker lacks epoch/generation tracking. If a long-running request starts during a `StateClosed` period, the breaker trips to `StateOpen`, recovers through `StateHalfOpen`, and returns to `StateClosed`, the ancient in-flight request can finally complete and erroneously record its result in the *new* `StateClosed` window. This "epoch pollution" can prematurely trip the breaker again based on stale failures from before the recovery.

## Context Cancellation RTT Pollution
**File:** `D:\CodingProjects\foundation\sync\limiter\limiter.go`
**Description:** In `Adaptive.Acquire`, if the context expires at the exact same moment that `Adaptive.Release` allocates a slot to the waiting channel, a race condition occurs. If `<-ctx.Done()` is selected over `<-ch`, the method searches `l.waitChs` for `ch`. Since `Release` already removed it, `found` is false. `Acquire` then correctly calls `l.Release(0)` to return the allocated slot. However, `Release` treats this `0` RTT as a valid `time.Nanosecond` (due to sanitization) and feeds it into the Vegas congestion avoidance algorithm. This drastically skews `minRTT` and `smoothedRTT` toward 1ns, artificially suppressing the dynamic limit and permanently crippling the limiter's throughput.
- **Bug in `sync/limiter/limiter.go`**: In the `Keyed` rate limiter, the background `sweepLoop` evicts limiters strictly based on `lastTouch` time, which is updated *before* waiting. If a request blocks in `Wait()` for longer than the configured `ttl`, the `sweepLoop` deletes the active limiter from the map. Subsequent requests for the same key will then instantiate a *new* limiter with a full token bucket, completely bypassing the ongoing rate limit wait queue and violating the rate limit guarantees.

## Semaphore O(W^2) Mass Cancellation Vulnerability
**File:** `D:\CodingProjects\foundation\sync\semaphore\semaphore.go`
**Description:** In `Semaphore.Acquire`, when a context expires, the method iterates over `s.waiters` to find the channel and removes it using `append(s.waiters[:i], s.waiters[i+1:]...)`. This requires shifting all subsequent elements, operating in `O(W)` time where `W` is the number of waiters. If a massive burst of requests is cancelled simultaneously (e.g., due to a client timeout or server shutdown), removing `W` waiters will take `O(W^2)` time, locking `s.mu` for increasingly long durations and completely freezing the semaphore during the cancellation storm.
- **Bug in `types/secret/secret.go`**: The documentation for `UnmarshalJSON` explicitly claims "It is safe for concurrent use." However, it takes a pointer receiver and mutates the internal `s.val` field directly via `json.Unmarshal` without any synchronization (e.g., mutexes). Concurrent unmarshaling into the same `Secret` instance will result in a data race and potential memory corruption, completely contradicting the documentation's safety guarantees.
## keylock.go Audit Report

### Vulnerabilities & Inconsistencies
1. **ForceUnlock Breaks Mutual Exclusion (Vulnerability):** 
   - `ForceUnlock` completely breaks mutual exclusion when other goroutines are waiting for the lock. 
   - When `ForceUnlock(key)` is invoked, it deletes `key` from `km.locks` and unblocks waiters on `ref.mu`.
   - The unblocked waiter subsequently proceeds into the critical section, believing it has acquired the lock safely. However, because the key is no longer in `km.locks`, a third concurrent caller to `Lock(key)` will create a completely new `refCounter` and acquire it instantly. 
   - Both the unblocked waiter and the third caller will execute their critical sections concurrently, defeating the core purpose of a lock. Additionally, the unblocked waiter will eventually panic upon calling `Unlock()` because the old reference no longer exists in `km.locks`. A reproduction test `TestReproduce_ForceUnlockBreaksMutualExclusion` has been added.

## Exponential Backoff Decreasing Delay Bug
**File:** `D:\CodingProjects\foundation\sync\backoff\backoff.go`
**Description:** In `NewExponential`, the fallback logic for an invalid or zero `max` duration assigns `max = 30 * time.Second`. However, if `initial` is greater than this fallback value (e.g., `initial = 60 * time.Second` and `max = 0`), the condition `max < initial` evaluates *before* `max` would hypothetically be updated to match `initial`. As a result, the `Exponential` backoff policy is created with `Initial=60s` and `Max=30s`. During `NextDelay`, the first attempt returns `60s`, but the second attempt is clamped to `Max`, returning `30s`. This causes the backoff delay to decrease, contradicting the exponential growth guarantee. A PoC test `TestReproduce_ExponentialBackoffDecreaseBug` has been added to `backoff_test.go`.
- **Bug in `text/casing/casing.go`**: The `convertDelimited` function used by `ToSnake`, `ToKebab`, and `ToScreamingSnake` fails to strip leading, trailing, and consecutive delimiters (spaces, dashes, underscores). This results in malformed casing output like `ToSnake("  hello  ")` returning `hello__` instead of the expected `hello`. This is highly inconsistent with `SplitWords` and `ToCamel` which correctly strip redundant delimiters.
- **Bug in `text/diff/drift.go`**: The `ExitCode` method in `Report` incorrectly fails the build (returns exit code 2) when ghost endpoints are detected, even if `Additive` mode is explicitly set to `true`. Additive mode is specifically designed to suppress ghost endpoints and allow them without failing, but `ExitCode` completely ignores the `Additive` field and checks only `GhostCount() > 0`.

## TryLock Pool Leak
**File:** `D:\CodingProjects\foundation\sync\keylock\keylock.go`
**Description:** In `TryLock`, if the inner `ref.mu.TryLock()` fails, the method proceeds to decrement `ref.count`. If a concurrent `Unlock` was executed by the current owner just before this decrement, `ref.count` can reach `0`. When this happens, `TryLock` correctly deletes the key from `km.locks`, but it fails to recycle the `refCounter` via `refCounterPool.Put(ref)`. This causes the pool to leak objects and forces unnecessary heap allocations under specific contention patterns. A PoC test `TestReproduce_TryLockLeak` has been added.
- **Bug in `types/values/values.go`**: The `NumberString` generic function fails to properly format custom floating-point types (e.g. `type MyFloat float64`). Because the type switch uses `any(v).(type)` instead of reflecting the underlying kind, custom types fall through to the `default` case. This casts the value to `int64(v)` and incorrectly truncates decimal values, formatting them as integers (e.g., `MyFloat(3.14)` becomes `"3"` instead of `"3.14"`).
## linkedlist.go Audit Report

### Vulnerabilities & Inconsistencies
1. **Capacity/Memory Leak on Free List Reuse:** 
   - When elements are removed from the list, they are correctly added to the `free` list for memory pooling.
   - However, in `alloc()`, when an element is subsequently popped from the `free` list, `l.allocs` is erroneously incremented.
   - Because `l.allocs` tracks the next unallocated index for the chunk arrays, incrementing it while reusing a previously allocated element causes the list to skip available index slots entirely.
   - Over time, heavily churning a list (repeated pushes and removals) will cause `l.allocs` to grow unboundedly, forcing the allocation of new `listChunk` arrays and leaking memory, completely defeating the purpose of the memory pool.
   - A reproduction test `TestReproduce_AllocLeakOnFreeListReuse` has been added to `bug_test.go` to explicitly verify and reproduce the bug.
- **Bug in `timekit/iso8601.go`**: The `AppendRFC3339` and `FormatRFC3339` functions assume all years are positive. When formatting a time with a negative year (e.g. `time.Date(-1, ...)`), the modulo operation `year % 10` returns a negative number (`-1`). This causes the byte arithmetic `byte('0' + year%10)` to underflow into invalid ASCII characters like `/` (e.g., `-1` formats as `000/`).

## Linked List Chunk Allocation Leak
**File:** `D:\CodingProjects\foundation\structures\linkedlist\linkedlist.go`
**Description:** In the `alloc()` method, elements popped from the free list (`l.free != -1`) incorrectly increment `l.allocs`. Since `l.allocs` is used as a monotonically increasing high-water mark for the chunk size boundaries, repeatedly adding and removing elements causes `l.allocs` to falsely blow past the chunk boundaries, abandoning perfectly good slots in existing chunks and allocating entirely new chunk arrays endlessly, leading to an unbounded memory leak. A PoC `TestReproduce_LinkedListMemoryLeak` has been added.

## dedup.go Audit Report

### Vulnerabilities & Inconsistencies
1. **Broken Deduplication via Unconditional Map Deletion:** 
   - In `run()`, when the worker goroutine completes (or panics), it unconditionally executes `delete(g.calls, key)`. 
   - However, if all waiting callers (including the initiator) cancel their contexts early, `wait()` will already delete the call from `g.calls`.
   - If a new `Do` request arrives for the same key *after* the cancellation but *before* the original worker finishes, a new call is registered in `g.calls`.
   - When the original worker finally finishes, its unconditional `delete` will completely obliterate the **new** active call from `g.calls`.
   - Subsequent requests for the same key will fail to find the active call and spawn duplicate redundant workers, entirely defeating the deduplication guarantees.
   - A reproduction test `TestReproduce_OldWorkerDeletesNewCall` has been added to `bug_test.go` to explicitly verify the vulnerability.

2. **Silently Swallowed Worker Panics:** 
   - In `run()`, if the worker panics, it attempts to distribute the recovered panic to the initiator via `c.initiator`.
   - If the initiator cancelled its context early, its channel is deleted from `c.waiters`.
   - As a result, the panic is never propagated back to the initiator or re-panicked. It is silently dropped, causing a critical loss of failure visibility. Secondary waiters receive `ErrWorkerPanicked`, but the actual panic trace is entirely swallowed.
   - A reproduction test `TestReproduce_PanicSwallowedOnInitiatorCancel` has been added to `bug_test.go` to verify the swallowed panic.
- **Bug in `silicon/bytesconv/bytesconv.go`**: The `ParseUintFast` function returns an `int64` and lacks overflow bounds checking. As a result, when parsing large valid unsigned 64-bit integer strings (e.g. `"9223372036854775808"`, which is `MaxInt64 + 1`), the value wraps around in two's complement and it returns a large negative number (`-9223372036854775808`) while returning `true` for success.

## event.go Audit Report

### Vulnerabilities & Inconsistencies
1. **Empty Subscription Goroutine/Resource Leak on Close:** 
   - When a user calls `Subscribe()` or `SubscribeBuffer()` without passing any event types, a valid `Subscription` is created and returned.
   - However, because the `evs` list is empty, the loop `for _, ev := range evs` is bypassed, meaning the subscription is NEVER added to the bus's internal registry (`b.subs`).
   - Consequently, when `b.Close()` is called, it iterates only over `b.subs` and `b.all` to close the active subscription channels. The empty subscription is completely ignored.
   - This causes the underlying channel `sub.ch` to remain open indefinitely, leading to a goroutine leak or a deadlock for any caller waiting on `sub.C()` expecting the bus shutdown to unblock them.
   - A reproduction test `TestReproduce_EmptySubscriptionLeakOnClose` has been added to `bug_test.go` to explicitly demonstrate the bug.
