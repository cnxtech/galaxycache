# galaxycache

galaxycache is a caching and cache-filling library, adapted from groupcache, intended as a
replacement for memcached in many cases.

For API docs and examples, see http://godoc.org/github.com/vimeo/galaxycache

## Summary of changes

Our changes include the following:
* Overhauled API to improve useability and configurability
* Improvements to testing by removing global state
* Improvement to connection efficiency between peers with the addition of gRPC
* Added a `Promoter` interface for choosing which keys get hotcached
* Made some core functionality more generic (e.g. replaced the `Sink` object with a `Codec` marshaler interface, removed `Byteview`)

We also changed the naming scheme of objects and methods to clarify their purpose with the help of a space-themed scenario:

Each process within a set of peer processes contains a `Universe` which encapsulates a map of `Galaxies` (previously called `Groups`). Each `Universe` contains the same set of `Galaxies`, but each `key` (think of it as a "star") has a single associated authoritative peer (determined by the consistent hash function). When `Get` is called for a key in a `Galaxy`, the local cache is checked first. On a cache miss, the `PeerPicker` object delegates to the peer authoritative over the requested key. The data is fetched from a remote peer if the local process is not the authority. That other peer then performs a `Get` to either find the data from its own local cache or use the specified `BackendGetter` to get the data from elsewhere, such as by querying a database.


### New architecture and API

* Renamed `Group` type to `Galaxy`, `Getter` to `BackendGetter`, `Get` to `Fetch` (for newly named `RemoteFetcher` interface, previously called `ProtoGetter`)
* Reworked `PeerPicker` interface into a struct; contains a `FetchProtocol` and `RemoteFetchers` (generalizing for HTTP and GRPC fetching implementations), a hash map of other peer addresses, and a self URL

### No more global state

* Removed all global variables to allow for multithreaded testing by implementing a `Universe` container that holds the `Galaxies` (previously a global `groups` map) and `PeerPicker` (part of what used to be `HTTPPool`)
* Added methods to `Universe` to allow for simpler handling of most galaxycache operations (setting Peers, instantiating a Picker, etc)

### New structure for fetching from peers (with gRPC support)

* Added an `HTTPHandler` and associated registration function for serving HTTP requests by reaching into an associated `Universe` (deals with the other function of the deprecated `HTTPPool`)
* Reworked tests to fit new architecture
* Renamed files to match new type names

### A smarter Hotcache with configurable promotion logic

* Promoter package provides an interface for creating your own `ShouldPromote` method to determine whether a key should be added to the hotcache
* Newly added Candidate Cache keeps track of peer-owned keys (without associated data) that have not yet been promoted to the hotcache
* Provided variadic options for `Galaxy` construction to override default promotion logic (with your promoter, max number of candidates, and relative hotcache size to maincache)


## Comparison to memcached

See: https://github.com/golang/groupcache/blob/master/README.md

## Loading process

In a nutshell, a galaxycache lookup of **Get("foo")** looks like:

(On machine #5 of a set of N machines running the same code)

 1. Is the value of "foo" in local memory because it's super hot?  If so, use it.

 2. Is the value of "foo" in local memory because peer #5 (the current
    peer) is the owner of it?  If so, use it.

 3. Amongst all the peers in my set of N, am I the owner of the key
    "foo"?  (e.g. does it consistent hash to 5?)  If so, load it.  If
    other callers come in, via the same process or via RPC requests
    from peers, they block waiting for the load to finish and get the
    same answer.  If not, RPC to the peer that's the owner and get
    the answer.  If the RPC fails, just load it locally (still with
    local dup suppression).

## Help

Use the golang-nuts mailing list for any discussion or questions.
