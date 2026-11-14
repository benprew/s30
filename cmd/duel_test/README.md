# Duel memory profiler

Build the graphical AI-vs-AI profiling harness:

```bash
make duelprofile
```

Run it without opening a visible window:

```bash
./dist/duel_profile \
  -headless \
  -autoplay \
  -duels 10 \
  -memprofile /tmp/duel-heap.pprof \
  -allocprofile /tmp/duel-allocs.pprof \
  -cpuprofile /tmp/duel-cpu.pprof
```

The harness prints Go heap, process RSS, garbage-collection, sprite-registry,
and card-image-cache counters every 600 frames and after every duel. Use
`-memstatframes` to change the interval. `-profileframes` supplies a total-frame
limit for shorter captures.

Builds with the `embedded_card_images` tag load embedded card art by default
to match normal game startup. Pass `-load-card-images=false` for an A/B run
that isolates its memory cost. Other builds fetch artwork on demand.
Linux without a display needs Xvfb even with `-headless`.

The default heap sampling rate keeps long profiling runs practical. Add
`-memprofilerate 1` only for a short, allocation-exact capture; writing that
profile can be slow and materially changes the program's memory behavior.

Inspect retained memory and cumulative allocations separately:

```bash
go tool pprof -http=:8080 ./dist/duel_profile /tmp/duel-heap.pprof
go tool pprof -http=:8081 ./dist/duel_profile /tmp/duel-allocs.pprof
```

Heap profiles cover Go-managed memory. RSS also reflects native and graphics
driver memory, although it cannot attribute those bytes to individual images.
Run without `-duel-log` for representative allocation results.

Run the graphical memory regression test:

```bash
make test-duel-memory
# Linux without a display:
xvfb-run -a make test-duel-memory
```

The test runs four completed AI-versus-AI duels in a separate process with
both seats using the X-spell deck. It uses fixed local card images to remove
network and artwork variation, and excludes the first duel as warm-up. Each
later duel must allocate at most 256 MiB and 2,000,000 objects. After garbage
collection, retained Go heap must stay within 16 MiB of the first duel and
live goroutines within four of that baseline. A three-minute timeout fails
stalled runs. CI runs this check under Xvfb.

These budgets allow for shuffled decks and different duel lengths. They catch
large regressions; they do not measure GPU memory or enforce exact allocation
counts. The fixed test images also exclude artwork download and decode costs.

For per-duel counters in a manual run, add `-memory-report /tmp/duel-memory.json`.
This forces GC between duels and records allocated bytes, allocation counts,
post-GC heap bytes, and goroutine counts. `-fast` raises the update rate to
1000 per second for shorter automated runs. Omit it when profiling normal
60-update-per-second behavior. A frame limit that stops a memory-report run
before all requested duels finish returns an error.
