# Changelog

## [0.2.2](https://github.com/Lemon4ksan/foundation/compare/v0.2.1...v0.2.2) (2026-10-17)


### Bug Fixes

* **silicon/sysnet:** resolve Windows compilation failure due to syscall.EWOULDBLOCK ([#12](https://github.com/Lemon4ksan/foundation/issues/12)) ([4f8d04d](https://github.com/Lemon4ksan/foundation/commit/4f8d04d6c2dc485b856b3380f24e8f97921cecc8))

## [0.2.1](https://github.com/Lemon4ksan/foundation/compare/v0.2.0...v0.2.1) (2026-10-17)


### Bug Fixes

* **encoding/json:** handle direct 1-word struct interfaces and optimize map encoder ([#10](https://github.com/Lemon4ksan/foundation/issues/10)) ([8f87d5a](https://github.com/Lemon4ksan/foundation/commit/8f87d5aa5c2216b82cd9495107a062ad8265cc0d))

## [0.2.0](https://github.com/Lemon4ksan/foundation/compare/v0.1.0...v0.2.0) (2026-10-14)


### ⚠ BREAKING CHANGES

* **async:** push iterators, test consolidation, and zero-alloc accessors
* **fskit:** concurrent push iterators, zero-alloc media heuristics, atomic file operations and test consolidation
* **urlkit:** zero-allocation fast-paths, query iterator, cacheline isolation and test consolidation
* **bufkit:** cache-coherence SPSC ring, streaming chain extensions and test consolidation
* **pathkit:** zero-allocation push iterators, caller-supplied Append forms and lexical operations
* **refkit:** zero-allocation overhaul with iter.Seq, SchemaCache and unboxed accessors

### Features

* **async:** push iterators, test consolidation, and zero-alloc accessors ([ace8439](https://github.com/Lemon4ksan/foundation/commit/ace84397d57f531e2a22fc471d5fd05d8423985b))
* **bufkit:** cache-coherence SPSC ring, streaming chain extensions and test consolidation ([91c9341](https://github.com/Lemon4ksan/foundation/commit/91c9341a44a8ee356078415bb41b22cb1834267a))
* complete v0.2.0 modernization and semantic bug remediation ([71c4874](https://github.com/Lemon4ksan/foundation/commit/71c48743a7f18ec2f42ee449dc85f77a8bfa3b8c))
* **encoding/proto:** baseline wire engine ([7c55d59](https://github.com/Lemon4ksan/foundation/commit/7c55d59f4d3d5dd13cbc744a3beca34f50f8a80b))
* **encoding/proto:** Grand Synthesis of zero-allocation Protocol Buffers wire engine ([4d70242](https://github.com/Lemon4ksan/foundation/commit/4d70242ec02661bb3df51093b9036fe07a593a08))
* **fskit:** concurrent push iterators, zero-alloc media heuristics, atomic file operations and test consolidation ([137d7eb](https://github.com/Lemon4ksan/foundation/commit/137d7ebff097cfb1af9e5f7dc1237396ccc0fc4e))
* **generic:** add Lazy[T], monad/slice helpers and harmonize repository standards ([#8](https://github.com/Lemon4ksan/foundation/issues/8)) ([7e7fb18](https://github.com/Lemon4ksan/foundation/commit/7e7fb184973ad1c52132d0d3d6fac18833004176))
* **net,iokit:** add dns wire/svcb codecs and vectored buffer queue ([a8ccfc7](https://github.com/Lemon4ksan/foundation/commit/a8ccfc7971e57cc710ac04abb3a87e3790d51cdc))
* **pathkit:** zero-allocation push iterators, caller-supplied Append forms and lexical operations ([e1c0707](https://github.com/Lemon4ksan/foundation/commit/e1c07078a0e44f85acf939f00209bf5feeba7025))
* **refkit:** zero-allocation overhaul with iter.Seq, SchemaCache and unboxed accessors ([14186cc](https://github.com/Lemon4ksan/foundation/commit/14186cc5d09cf906ad780663ae9775680e6857b2))
* **silicon:** add zero-alloc hexkit primitives and SIMD text kernels ([8698726](https://github.com/Lemon4ksan/foundation/commit/86987264fd1cc7e56a4c90b72290dc42ca173981))
* **urlkit:** zero-allocation fast-paths, query iterator, cacheline isolation and test consolidation ([fdc5725](https://github.com/Lemon4ksan/foundation/commit/fdc572583124166917e708fad5579ffbb4f5a56b))


### Bug Fixes

* **alloc:** ADR-0003 capacity retention in bufkit, generic.SlicePool, breaker, limiter, semaphore, iokit ([573ffcc](https://github.com/Lemon4ksan/foundation/commit/573ffcce21e37c1b4cf5fa3c73b015b8daed766a))
* **alloc:** ADR-0003 Deque.ShrinkToFit, RequestArena tail 0-alloc, SPSC 128B prefetch false sharing elimination ([8b27d90](https://github.com/Lemon4ksan/foundation/commit/8b27d902b431260142d5c852b0c49fe7386af3e4))
* **alloc:** ADR-0003 linkedlist, minheap, and offheap zero-alloc and memory retention fixes ([cf0b7f5](https://github.com/Lemon4ksan/foundation/commit/cf0b7f51ee36a771a7e2f13220ce05572f0a2a6e))
* **all:** resolve all audit defects and establish clean regression test suite ([07581da](https://github.com/Lemon4ksan/foundation/commit/07581da82c2988a0108e9a93cfb8181db7b2d66b))
* **all:** resolve critical architectural bugs from Swarm audit ([f222b33](https://github.com/Lemon4ksan/foundation/commit/f222b33e095b01c56f54b87af4df19477ad45ed4))
* **arch:** decouple hexkit from cpukit and calibrate coverage baseline for Linux CI ([8758e13](https://github.com/Lemon4ksan/foundation/commit/8758e13a3607b322d13ff0ad241f22d6a1ab3df1))
* **boundary:** resolve parser hangs, depth bomb, integer overflows, and validation edge cases ([b49c7c0](https://github.com/Lemon4ksan/foundation/commit/b49c7c034d0de67c7ca1eb0da608d8e67a442a0d))
* **boundary:** resolve Unicode escapes, trailing garbage, URL null injection, SSRF authority ambiguity, and extreme year overflows ([71c245b](https://github.com/Lemon4ksan/foundation/commit/71c245b4039423f831e06e7de88c83e83f376897))
* **encoding/json:** update internal package imports and fuzz script ([e25f33f](https://github.com/Lemon4ksan/foundation/commit/e25f33ff748ad6486ff6bd4822273c2ba42e4d31))
* intercept -h and --help flags in app router to prevent subcommand ([9f9971e](https://github.com/Lemon4ksan/foundation/commit/9f9971e328332627de012f1d8da1a25d85f9f7ba))
* **mutator:** fix integer overflows in bin reader, nil pointer panics, varint peeker, and dns bounds checks ([4059661](https://github.com/Lemon4ksan/foundation/commit/4059661d6df8aeeb6b34089a93f3d7fea37e9856))
* **mutator:** fix pointer bombs in dns wire, svcb param validation, varint nil reader, and base64 length overflow ([f0111f0](https://github.com/Lemon4ksan/foundation/commit/f0111f0261ebf4d537b96e52641a2fcbffead24e))
* **pool,sign:** eliminate TestPool_ScaleDown timing flake and boost sign coverage to 96% ([ef5ae6f](https://github.com/Lemon4ksan/foundation/commit/ef5ae6fa556ec593834a84605d59a7b17ae3ef5d))
* **qa:** deduplicate coverage profile blocks, fix urlkit IPv6 zone fuzz discrepancy, and raise test coverage ([66b6b56](https://github.com/Lemon4ksan/foundation/commit/66b6b569129c162b0472482805a6a22d70226b69))
* resolve critical bugs found by AI Audit Swarm ([60ee8b0](https://github.com/Lemon4ksan/foundation/commit/60ee8b0fbdba88009063fd1640a9679ab9c2a71b))
* resolve final audit findings in parsing, SIMD, and tests ([6de6318](https://github.com/Lemon4ksan/foundation/commit/6de63181630574d76638dca63e700990fde72171))
* resolve linkedlist out-of-bounds array allocation and remove dead simd assembly ([0790538](https://github.com/Lemon4ksan/foundation/commit/0790538cc6755b57c3e115b9bf69fbb900df3620))
* resolve linter, typecheck, and formatting issues post-merge ([2c9532e](https://github.com/Lemon4ksan/foundation/commit/2c9532e97887665a2734f9fe12ba2cb4dc4cfa9c))
* **scripts:** sync fuzz_all with active targets (add encoding/proto, drop removed pkgs) ([033944f](https://github.com/Lemon4ksan/foundation/commit/033944f9500cf9d0e46f538e03a5cf2850b1fc13))
* **silicon/clock:** calibrate realistic frequency and resolution on ARM64 macOS ([8c19a83](https://github.com/Lemon4ksan/foundation/commit/8c19a83780a73d26dadaba532ea93ef0a5f771fb))
* **silicon/pool:** eliminate dead code in Alloc and achieve 100% deterministic coverage ([b88a394](https://github.com/Lemon4ksan/foundation/commit/b88a394de14b0ab280566ff4e4a2495d71d45b52))
* **state_breaker:** resolve AEAD framing attacks, Ed25519 signature malleability, Argon2id bounds, and secret serialization leaks ([789335d](https://github.com/Lemon4ksan/foundation/commit/789335d89a6c833d48b06aac3e80fcb6e389365d))
* **state_breaker:** resolve slice bounds panic in PBKDF2, OpenSSH fingerprint compatibility, short nonce panic, secret wiping, and FSM invariants ([d4238bd](https://github.com/Lemon4ksan/foundation/commit/d4238bd49100bfe301e743a64133f50faa2afa86))
* **types/secret:** prevent value receiver copy data race on concurrent Expose and Wipe ([0a66081](https://github.com/Lemon4ksan/foundation/commit/0a660816e29186fd40e788d71f3ab75d60b60e18))


### Performance Improvements

* **escape:** eliminate interface boxing, closure escapes, and tag allocations ([acec612](https://github.com/Lemon4ksan/foundation/commit/acec612f07809f8b6cad3929aacd06b663374bab))
* **escape:** lock-free atomic SchemaCache, inlined Tag.Value, and clock duration arithmetic ([770ff9e](https://github.com/Lemon4ksan/foundation/commit/770ff9ec13d8cfe8b2c19b54e0d2e5798eff1054))
* **escape:** zero-alloc ParseHTTPDate, UUID unmarshal/formatting, and values serialization ([b49d315](https://github.com/Lemon4ksan/foundation/commit/b49d315f78696a9cf16dd1ec5d7d7cad4f77b998))
* **nano:** 32-byte SWAR IndexByte/ValidUTF8, 64-bit store hex packing, and deque unsigned bounds folding ([3e7eb7d](https://github.com/Lemon4ksan/foundation/commit/3e7eb7de0d5dcc4d0b1806f6b48cd1e780033637))
* **nano:** RadixTree direct node traversal & BCE, cpukit branchless caching, and endian BCE hoisting ([82e8cc9](https://github.com/Lemon4ksan/foundation/commit/82e8cc91e90a5ebed21af234345672fbf6dfea96))
* **nano:** SWAR, BCE elimination, zero-alloc casing and json decode optimization ([f4e28c6](https://github.com/Lemon4ksan/foundation/commit/f4e28c6f4d5cb5fb74c2aace8b3e9e8df6099db1))


### Miscellaneous Chores

* **ci:** configure release-please manifest mode and set 0.2.0 ([cfbcae0](https://github.com/Lemon4ksan/foundation/commit/cfbcae0cbeb9b1f5c23c80514a50f7bea3e50f65))

## [0.1.0](https://github.com/Lemon4ksan/foundation/compare/v0.0.1...v0.1.0) (2026-10-03)


### Features

* add base64 encoding from aoni ([9340fc7](https://github.com/Lemon4ksan/foundation/commit/9340fc7fbb8eacff51357f34a498b3d1e95b70c6))
* add either monad ([8dd13c5](https://github.com/Lemon4ksan/foundation/commit/8dd13c5c72b781d1d5a5800cb5b644533b9ecece))
* add gfni simd instruction ([7c37a42](https://github.com/Lemon4ksan/foundation/commit/7c37a42d350438abf6dbc72602326b0f98fea9ff))
* add hex package ([425a8cd](https://github.com/Lemon4ksan/foundation/commit/425a8cdf4734671e64861f4176ffa01a0608a8ec))
* add net icmp and tcp, update readme ([c072200](https://github.com/Lemon4ksan/foundation/commit/c072200a4049b3fd87a1cda5a31e084f6f77d566))
* add QUIC and Headkit modules from mach ([0a50e83](https://github.com/Lemon4ksan/foundation/commit/0a50e839345cd1d04b0065970cd6c95b69214de2))
* add vfs, fskit (fastwalk, mmap, media), and entropy analysis to foundation ([5b315f8](https://github.com/Lemon4ksan/foundation/commit/5b315f86afdb66c24148d47733782401ea1fa670))
* **async/context:** implement fast Context flat-array context with ([f66d69a](https://github.com/Lemon4ksan/foundation/commit/f66d69a22bf21b723bc28319ffc7775af74c5070))
* **async/rate:** add Token Bucket rate limiter eliminating x/time dependency ([9874724](https://github.com/Lemon4ksan/foundation/commit/9874724ebb1d741cfc618279f307fd1c47ad5940))
* **borrow:** add Bytes() slice accessor alias to borrow.Bytes ([f60cbbc](https://github.com/Lemon4ksan/foundation/commit/f60cbbc0a601345f784f9a4a9fda63951dca6ffd))
* **borrow:** add linear ownership, zero-alloc borrows, generational slots, and scoped arenas ([b881853](https://github.com/Lemon4ksan/foundation/commit/b8818538e6b84cc3a7cab69770bc00d4bbd4a90e))
* **borrow:** add NewScope alias for AcquireScope ([640e47e](https://github.com/Lemon4ksan/foundation/commit/640e47e13bf796080ca6a863a6e95f439d722271))
* **borrow:** add Slice method to Bytes handle ([bea6225](https://github.com/Lemon4ksan/foundation/commit/bea6225a4a35eee48b26bbabaf79af6978e21966))
* **c2plan9:** support AArch64 NEON architecture and cross-disassembly ([42d48a9](https://github.com/Lemon4ksan/foundation/commit/42d48a924c5a7175aa10330602acba4bad54c56e))
* **c2plan9:** support rodata constant extraction and dynamic stack frame sizing ([9325d52](https://github.com/Lemon4ksan/foundation/commit/9325d525f3e2bf579699a046206cf970144d2b39))
* **clock:** implement Hardware RDTSC assembly instruction, TSC calibration and CoarseNowUnix ([2e8ad8c](https://github.com/Lemon4ksan/foundation/commit/2e8ad8cc766ce78d3234cab6b934ae9141229548))
* **codec/json:** support anonymous embedded structs and non-string map keys ([4af038b](https://github.com/Lemon4ksan/foundation/commit/4af038bfa97610551a6333d3302c785e54106049))
* **codec:** add zero-alloc LZ4 engine and byte-shuffle filter ([090234a](https://github.com/Lemon4ksan/foundation/commit/090234ae2c2465fdb5e8a943820a094161aa6530))
* **codec:** introduce compress (gzip, brotli, zstd, lzma2, deflate) and byte stream filters (bcj, delta) ([f6b9e0d](https://github.com/Lemon4ksan/foundation/commit/f6b9e0d8512b94f3d1e97dd5a89fe489be544267))
* **core:** add RFC protocols, TLS primitives, HTTP codecs, and types ([baa3e4e](https://github.com/Lemon4ksan/foundation/commit/baa3e4ede6ea51eb1ce6bb6d4c20ea14a181c8df))
* **core:** consolidate standard RFC codecs, AST, diff, and network foundations ([42856e0](https://github.com/Lemon4ksan/foundation/commit/42856e01b6680fe177daa08e038b602f62c27422))
* **cryptokit:** add comprehensive cryptokit suite and move rand to randkit ([0570cea](https://github.com/Lemon4ksan/foundation/commit/0570cea98b6a411f3bfe059d9f0629d47b10372a))
* **cryptokit:** add GF(2^8) Galois field arithmetic and Shamir's Secret Sharing (M-of-N) ([62813ed](https://github.com/Lemon4ksan/foundation/commit/62813ed0a72d402e9e0f170c16167919d23c98c9))
* **dns:** add zero-alloc LookupNetIP and LookupFirstNetIP using netip.Addr ([ab44ec0](https://github.com/Lemon4ksan/foundation/commit/ab44ec05d262ea27a51595fefedd06b78fdd8f9d))
* **event:** add iter utils ([4230cc6](https://github.com/Lemon4ksan/foundation/commit/4230cc6ff7f43d3f3c6fb5ec86363e9a84bd3336))
* **foundation:** achieve comprehensive test coverage and byte-level alignment ([8a84317](https://github.com/Lemon4ksan/foundation/commit/8a843172a043b5918b219cb3caa190d65d4d9776))
* **foundation:** add timekit, bufkit, stream and header utilities ([e7d680b](https://github.com/Lemon4ksan/foundation/commit/e7d680bdb094c73ba7ff0177af73279d2375e8ca))
* **generic:** add Filter helper with slice allocation ([45a247a](https://github.com/Lemon4ksan/foundation/commit/45a247ad6562f154053702b4add6c8ed8ff331b3))
* **generic:** add GetOrSet to ShardedMap ([4c5717e](https://github.com/Lemon4ksan/foundation/commit/4c5717e2cac3adb7efe4f552c2d9682dfc3460b9))
* **generic:** add Mutex[T] container, TryWith, and WithGuard / WithRGuard lock helpers ([43e4008](https://github.com/Lemon4ksan/foundation/commit/43e40082e9549939ddb06badf1c35347d5ff8b68))
* **generic:** add RaceFirstSuccess structured concurrency helper ([b9377c4](https://github.com/Lemon4ksan/foundation/commit/b9377c4f344a1b8899ee92867e589ea220d5a4eb))
* **generic:** add Safe, Atomic, Signal, Pool, ConcurrentMap, and Singleflight primitives ([251c324](https://github.com/Lemon4ksan/foundation/commit/251c324fcfeb3aad6db8cffed39d4c6c95ff1cf3))
* **generic:** add Values() method to LRU ([5f06c04](https://github.com/Lemon4ksan/foundation/commit/5f06c0498f05f180046ae3270e54fc10f88717c5))
* **generic:** upgrade ShardedMap with maphash, cache-line padding, and non-blocking TryGet/TrySet ([dd9b550](https://github.com/Lemon4ksan/foundation/commit/dd9b550ce77ffdaf17ae18e5b84b1453693689b5))
* handle nil pointer marshaling and non-blocking handshake ([c7a65ba](https://github.com/Lemon4ksan/foundation/commit/c7a65ba13aebce55728a50e2d19627f2ab96a91e))
* **headkit,io,rand,generic:** add BaseMediaType, StripBOM, Wyrand RNG and ToResult monads ([4306f5a](https://github.com/Lemon4ksan/foundation/commit/4306f5ae8e91a2074762956b4d00973889c4bc92))
* **hpack:** migrate high-performance zero-allocation HPACK from mach ([5d3b342](https://github.com/Lemon4ksan/foundation/commit/5d3b342e40d0aae5dc826d5970c84b6d5323df6e))
* implement full RFC 9204 QPACK dynamic table ported from Chromium ([aa2c81d](https://github.com/Lemon4ksan/foundation/commit/aa2c81ddf10f381e4fa066b0e29799abaf3a47f5))
* **io:** add high-performance zero-allocation streaming and replayable body primitives ([1603f78](https://github.com/Lemon4ksan/foundation/commit/1603f78831bccf714e3a9ab9bae5039413b7490d))
* **kms:** remove AWS KMS client and provider specifics ([8d7ef89](https://github.com/Lemon4ksan/foundation/commit/8d7ef892f5717b96e40cf38b29b1482a43ce4582))
* **logkit:** add MaxStringLen to prevent runaway log lines and truncate overly long string/error fields ([09beb07](https://github.com/Lemon4ksan/foundation/commit/09beb07cf54a147a89e399b07d66281bd9b84ff8))
* **net/dns/wire:** add RFC 1035, RFC 6891 EDNS0, and RFC 9460 HTTPS/ECH binary wire codec ([896f611](https://github.com/Lemon4ksan/foundation/commit/896f611a2c5ec5ffbc510f028e18097d7ca2533c))
* **net/hpack:** implement zero-allocation HPACK decoder and encoder with Decoder.Reset support ([5baff5c](https://github.com/Lemon4ksan/foundation/commit/5baff5c302455d028d1d85a2ba9555609202ff5e))
* **net/idna:** implement self-contained IDNA2008 and Punycode (RFC 3492/5891) codec ([49a672d](https://github.com/Lemon4ksan/foundation/commit/49a672d94ec59fda37c49c0c16b4648dd2880239))
* **net/ip:** add lock-free IP rotator, IPv6 subnet generator, and private IP matcher ([0ee33db](https://github.com/Lemon4ksan/foundation/commit/0ee33dbeea5cd886b4d4a8dafa3f2055df604b9a))
* **net/proxy:** add pure-Go SOCKS5, Direct and PerHost proxy router ([0b6c147](https://github.com/Lemon4ksan/foundation/commit/0b6c1473546ac9ba726aa727ac64c0d27412d87f))
* **net/psl:** implement zero-allocation Public Suffix List and eTLD+1 byte matcher ([d08d30f](https://github.com/Lemon4ksan/foundation/commit/d08d30f36bc2bb92ded3a5b85232a9e10d4b4a9e))
* **net/url:** add hardware CRC32 sharded URL cache and zero-allocation query tools ([1d4b8dc](https://github.com/Lemon4ksan/foundation/commit/1d4b8dceac5c21087c75d6e40d61bea3cb291513))
* **net:** extract qpack from mach to foundation ([46e3fdb](https://github.com/Lemon4ksan/foundation/commit/46e3fdb84c15ecdc2bc0db1d3853740e29105e69))
* **pathkit,headkit,refkit:** add Path abstraction, HTTP header engine, ([9a7cda3](https://github.com/Lemon4ksan/foundation/commit/9a7cda35e3bffcbbcb165f2c42a2db925c138a32))
* **refkit:** add string serialization ([ae415cd](https://github.com/Lemon4ksan/foundation/commit/ae415cdfd72d037e734b06bdccf67e33024bf751))
* **silicon/endian:** add Little-Endian and Big-Endian (including 24-bit) zero-copy primitives ([07c203c](https://github.com/Lemon4ksan/foundation/commit/07c203cc5de106037d8b4b2bca20d1ce31c1e379))
* **silicon/sysnet:** add cross-platform socket tuning, vectorized I/O, and batch UDP ([7c12531](https://github.com/Lemon4ksan/foundation/commit/7c1253134c8ac18aaae8374e98531f43913d4951))
* **silicon:** add adaptive read strategy, vectored IO buffer queue, and fastscan ([5045ab0](https://github.com/Lemon4ksan/foundation/commit/5045ab03795ba36b3e652a2fbf79eb62db7dd649))
* **silicon:** add AVX2 kernels for casing, base64, and url percent decoding ([57c79ad](https://github.com/Lemon4ksan/foundation/commit/57c79ad306206327a1f67edd52235b68ee3ed99d))
* **silicon:** add universal cpukit and endian packages ([5e55402](https://github.com/Lemon4ksan/foundation/commit/5e554022cbacd9453889f7cd629848162148265d))
* **silicon:** implement c2plan9 compiler and LLVM AVX2 kernel layer ([1c3e07b](https://github.com/Lemon4ksan/foundation/commit/1c3e07b72ce33a6ae6f2a88ed1974f67fd68ae36))
* **silicon:** move csrc to root and accelerate hex, uuid, and json with AVX2 ([301a87f](https://github.com/Lemon4ksan/foundation/commit/301a87fe1b998175b50e5523b9000424ff7398d4))
* **simd:** add ValidUTF8 accelerated with 64-bit SWAR vector processing ([e4fdd16](https://github.com/Lemon4ksan/foundation/commit/e4fdd16d7081d29f0e6e5047abee1a92f694bb7d))
* **spinlock:** implement 3-tier Adaptive Silicon Spinlock with TTAS, CPU PAUSE and CacheLinePad ([95b4647](https://github.com/Lemon4ksan/foundation/commit/95b4647d0b556b0a8fe11bd2b0a0ce51d5e90b72))
* **structures,encoding:** import core utilities from mach engine ([1a1944e](https://github.com/Lemon4ksan/foundation/commit/1a1944ea5482ec516eb9832bd2ba2765a89075e3))
* **sysnet:** add UDP GSO and GRO hardware socket offload support ([aa85cd3](https://github.com/Lemon4ksan/foundation/commit/aa85cd3f8493ed777a4427a4440c5dad5f3fe701))
* **testkit:** add AnError and polymorphic InDelta support ([d43a8dc](https://github.com/Lemon4ksan/foundation/commit/d43a8dcf03cecba8143e3109ed9d04724fa82b48))
* **testkit:** add complete *f assertion formatters in assert and require ([dbb5234](https://github.com/Lemon4ksan/foundation/commit/dbb5234a1039f9988c664d7cc0429112ddf1e39e))
* **testkit:** add complete assert and require API compatible with testify ([26686cb](https://github.com/Lemon4ksan/foundation/commit/26686cb46aba20606d53ac295ef00f70320c35ff))
* **testkit:** add zero-dependency testing toolkit with assertions, require, and GoMock engine ([b8e9dce](https://github.com/Lemon4ksan/foundation/commit/b8e9dced8ed88d73fc80375cd2df386409c1789f))
* **testkit:** support exported struct field comparison in assert.Equal ([8f56ec8](https://github.com/Lemon4ksan/foundation/commit/8f56ec8c1084d841c3164cc05e0305fee937d11c))
* **text:** implement zero-allocation web encodings, charmaps, and ([93683f8](https://github.com/Lemon4ksan/foundation/commit/93683f87d9c1e1e6f1b6f2c5e76eec8bc9736e8a))
* **tui:** add progresbar ([ff74e61](https://github.com/Lemon4ksan/foundation/commit/ff74e61ef9fb0c21e2a041b70dc9460113d04474))


### Bug Fixes

* add doc.go file so that the benchmarks can be located ([dc43867](https://github.com/Lemon4ksan/foundation/commit/dc438670455e28754a597222f5c4fed31d3baea4))
* **arm64:** fix rdtsc_arm64 assembly instruction and clean up simd_arm64 ([0d5a1f2](https://github.com/Lemon4ksan/foundation/commit/0d5a1f2711d4b374a166427e8352d3004dc5193d))
* ast json and url paths ([3138d4a](https://github.com/Lemon4ksan/foundation/commit/3138d4a24c98a3f61a4bbfe737f830b5dc756657))
* **auth:** RFC 7616 and RFC 7617 compliance for Basic and Digest Auth ([c2a4ea0](https://github.com/Lemon4ksan/foundation/commit/c2a4ea0a5ed118bc096c323028eb4e5156f7b452))
* **borrow:** prevent readers underflow and generation races in Slot ([648ed40](https://github.com/Lemon4ksan/foundation/commit/648ed40b520785d35304fe7474a5b75862b1b98a))
* **c2plan9:** support 32-bit and 64-bit BSWAP instruction width translation ([bdd6217](https://github.com/Lemon4ksan/foundation/commit/bdd6217ad401feb6b46ce6f522fc88f831836c2a))
* **c2plan9:** support 64-bit CMOVQ mnemonics and precise 8-bit register operand decoding ([18c34ed](https://github.com/Lemon4ksan/foundation/commit/18c34edd5f3ed5229f96257cbf2e7dd92293b327))
* **ci:** compile golangci-lint from source with Go 1.27 ([113bb2b](https://github.com/Lemon4ksan/foundation/commit/113bb2b8f8616f18a3de4fd6a3af86cb3fbf0989))
* **ci:** downgrade golangci-lint-action to v6 and fix pathkit cross-platform test ([e2b5e95](https://github.com/Lemon4ksan/foundation/commit/e2b5e956fe1c4671b8add4c3b96b8d636349e45a))
* **ci:** fix gci formatting in bufkit test ([02bf2f0](https://github.com/Lemon4ksan/foundation/commit/02bf2f04d3f97ccc1f9f6b626019561fa06ed83c))
* **ci:** restore golangci-lint-action to v7 latest ([6e30cdb](https://github.com/Lemon4ksan/foundation/commit/6e30cdbb98c55c7deabed3a21f626c424cb45fd6))
* **codec/entropy:** resolve internal test namespace ([59f1419](https://github.com/Lemon4ksan/foundation/commit/59f1419e69b5682e549b0500ee2caec4c298166c))
* **contentdisposition:** normalize Windows path separators for cross-platform filename sanitization ([514aced](https://github.com/Lemon4ksan/foundation/commit/514aced7e068527951e56052add4104a500e478d))
* context panic with type any ([a0c6e94](https://github.com/Lemon4ksan/foundation/commit/a0c6e94cc4e46ea508b3e07d117e9f2e753c89f0))
* **cookie:** enforce strict RFC 6265 max-age and path validation ([7530dba](https://github.com/Lemon4ksan/foundation/commit/7530dba0e76ff90dad1fc572fc5b0dcb5a91e103))
* **cryptokit/shamir:** avoid no-op tamper test when CRC ends in zero ([e8ac18a](https://github.com/Lemon4ksan/foundation/commit/e8ac18a42e81846748b569d30e5e21fb3f38590a))
* darwin compilation ([90a271d](https://github.com/Lemon4ksan/foundation/commit/90a271d07e4623c2600a058c3244d7ec128ab0d3))
* **encoding:** correct stale import paths in varint tests ([653d0ba](https://github.com/Lemon4ksan/foundation/commit/653d0ba7681b5bec250bff9b059b15554a9b779c))
* errorlint warning ([6abfc7c](https://github.com/Lemon4ksan/foundation/commit/6abfc7c0c2c13464275e65e072695981df53cfd8))
* **flate:** guard wildcopy bounds in dictDecoder near window end ([a5a8cb4](https://github.com/Lemon4ksan/foundation/commit/a5a8cb48d00a053105c99e888a5adc06bc11f9d0))
* **headkit:** handle escaped quotes in directive parsers ([d262d13](https://github.com/Lemon4ksan/foundation/commit/d262d13f1cd172a5493fcb60e9d059bb694eceb9))
* **hpack:** initialize default MaxCapacity to 4096 per RFC 7541 §4.2 ([7709c68](https://github.com/Lemon4ksan/foundation/commit/7709c688b2d7bf4c67dc6011b488cc806d974e34))
* **json:** invalide utf mismatch ([d88efac](https://github.com/Lemon4ksan/foundation/commit/d88efac513fcef1629d5f1efd6ed552debc1b23f))
* **json:** support recursive type definitions without stack overflow via lazy resolver ([a2182de](https://github.com/Lemon4ksan/foundation/commit/a2182dee2de400aab37cabddf4ad9a623625305e))
* **json:** support unmarshaling into existing interface pointer ([d1144fe](https://github.com/Lemon4ksan/foundation/commit/d1144fe7221d1345ce59d9b163e1521b67fa5bf5))
* **keylock:** check held status in KeyMutex.Unlock to prevent fatal crash ([71442bb](https://github.com/Lemon4ksan/foundation/commit/71442bb574bb247230e0806526054fb92ba71310))
* **limiter:** prevent panic on zero limit and slot leaks on cancelled acquire ([32bbf47](https://github.com/Lemon4ksan/foundation/commit/32bbf477cab1db7d023f21b6972d0d91a9124f2d))
* **lz4:** remove unused errors import in reader.go ([ac426cb](https://github.com/Lemon4ksan/foundation/commit/ac426cb0e28950d687d80d29fdbc6e2efacd1845))
* **lzma2:** decouple ResetStateKeepDict from dictionary history and ensure property header transmission ([4963cf7](https://github.com/Lemon4ksan/foundation/commit/4963cf71412106fe5abeb4fd2e258bdf0dcac14e))
* **net/quic:** restore valid wiki URL in UDP buffer size warning ([d46a891](https://github.com/Lemon4ksan/foundation/commit/d46a891f7602cc551aacd40aea4945ee7bbc0017))
* **offheap:** prevent integer overflow and truncation in Arena.Alloc ([2437446](https://github.com/Lemon4ksan/foundation/commit/2437446f51e67ed452168475f2082de45d4dad3b))
* **pool:** prevent integer overflow in RequestArena allocation size ([8214d58](https://github.com/Lemon4ksan/foundation/commit/8214d58cd2debb64075084fe33c12406b31ec837))
* **quic,argkit:** handle background packet packing during teardown and fix wiki URLs ([121e9f1](https://github.com/Lemon4ksan/foundation/commit/121e9f1d1f9396796feb28b48194510c37dd8ba9))
* **quic/varint:** use bitwise complement for MaxUint64 in tests ([64e4a06](https://github.com/Lemon4ksan/foundation/commit/64e4a06ea1789d40fba206e7fdd03588bec56b28))
* **quic:** eliminate data race on cancelErr in ReceiveStream ([e45b0d2](https://github.com/Lemon4ksan/foundation/commit/e45b0d2b7cf0b601a90f110a1dcd53a604e7d5b3))
* resolve gci formatting and nilerr issues in example tests ([b7124a7](https://github.com/Lemon4ksan/foundation/commit/b7124a7939afc29f98ab44ba1cf8e929dfd82357))
* **ringbuf:** prevent infinite bitshift loop in NewRingBuffer on negative capacity ([3ce2171](https://github.com/Lemon4ksan/foundation/commit/3ce21719568e308fc4e0645d3dcb5a5bb20fda37))
* **semaphore:** prevent slot leak on cancelled acquire and active underflow ([24b1be4](https://github.com/Lemon4ksan/foundation/commit/24b1be4125f1df4a24f44d0d69a53dbdb3a30962))
* **simd,c2plan9:** resolve macOS ARM64 fallback and test package naming ([e75a060](https://github.com/Lemon4ksan/foundation/commit/e75a06060f5f7e9f8994cdb122fc317254bee98c))
* **simd,json:** fix fastscan and json stream decoder errors ([9cbb471](https://github.com/Lemon4ksan/foundation/commit/9cbb471ddab6374a68037158d5d07512971184d1))
* **simd:** consolidate vector index and fastscan declarations across build tags ([e8180c0](https://github.com/Lemon4ksan/foundation/commit/e8180c01cc15e9c27672544800995252a59c379f))
* **simd:** consolidate vector scan declarations and formatting ([5cb5f4c](https://github.com/Lemon4ksan/foundation/commit/5cb5f4ce013446561ce7b7272940fd2b9c69e936))
* **simd:** isolate experimental archsimd behind goexperiment.simd and provide AVX2 plan9 fallback ([de4763a](https://github.com/Lemon4ksan/foundation/commit/de4763a9c37659b1ede184bcfebea179be4ea27a))
* **simd:** remove unused experimental archsimd import ([2ca5a23](https://github.com/Lemon4ksan/foundation/commit/2ca5a235ff1c85ee70ba2e7deef0d75567bf7b52))
* **sysnet:** rename p0f_darwin.go to p0f_bsd.go to fix build on FreeBSD and OpenBSD ([afebb6e](https://github.com/Lemon4ksan/foundation/commit/afebb6ee301b3dd570fa7955e5ee025a1b3e8b01))
* **testkit:** use assert.EqualExportedValuesf in require.EqualExportedValuesf ([6a91c26](https://github.com/Lemon4ksan/foundation/commit/6a91c26668c12219ef0c880fb934ea6c890929fb))
* **test:** resolve final GC leak in deque using go:noinline helper ([414c243](https://github.com/Lemon4ksan/foundation/commit/414c24391911cfc89ec2efadd03c51266720189c))
* **test:** resolve memory leaks in deque and strict allocation counts in foundation ([bb810ec](https://github.com/Lemon4ksan/foundation/commit/bb810ec9408be5414425fc24702516e935b51538))
* translation of unscaled load/store instructions ([8a2b4fd](https://github.com/Lemon4ksan/foundation/commit/8a2b4fd3c369cb8455f76a71f4f760201c76ed85))
* **urlkit:** address RFC 3986 and RFC 6874 violations ([f200add](https://github.com/Lemon4ksan/foundation/commit/f200add531cee229b9e67fe84da3d53bbacdf392))
* **urlkit:** RFC 3986 and RFC 6874 compliance fixes ([0b2fa28](https://github.com/Lemon4ksan/foundation/commit/0b2fa28388e9554474b8570c1de6487c46101aa1))


### Performance Improvements

* **bcj:** accelerate filterX86 with SWAR trailing-zeros instruction skip ([1b1a7a6](https://github.com/Lemon4ksan/foundation/commit/1b1a7a66186b7b9af5b771cd2895fa20da8b5b77))
* **lzma2:** scope worker match finder to blockSize and reuse worker cores ([255c99c](https://github.com/Lemon4ksan/foundation/commit/255c99ca743c1f89cb305ad90d21a5d2760f38cc))
