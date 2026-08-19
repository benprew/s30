# Duel memory profiler

Build the graphical AI-vs-AI profiling harness with embedded card art:

```bash
make duelprofile
```

On Linux, run it without a visible window through Xvfb:

```bash
xvfb-run -a ./dist/duel_profile \
  -duels 10 \
  -memprofile /tmp/duel-heap.pprof \
  -allocprofile /tmp/duel-allocs.pprof \
  -cpuprofile /tmp/duel-cpu.pprof
```

The harness prints Go heap, process RSS, garbage-collection, sprite-registry,
and card-image-cache counters every 600 frames and after every duel. Use
`-memstatframes` to change the interval. `-profileframes` supplies a total-frame
limit for shorter captures.

Embedded card art is loaded by default to match normal game startup. Pass
`-load-card-images=false` for an A/B run that isolates its memory cost.

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
