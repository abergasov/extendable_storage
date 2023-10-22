## Overview
run tests
```shell
make test
```

run linter
```shell
make lint
```

You've decided to create a competitor to Amazon S3 and you know how to make a better file storage service. 

When a file is sent to server A via REST, it needs to be split into approximately 6 equal parts and saved on storage servers Bn (where n ≥ 6). 

When there's a REST request to server A, you should retrieve the pieces from servers Bn, concatenate them, and return the file.

### Implementation
1. The system comprises 3 independent entities: the `storage node`, `orchestrator`, and `receiver`. 
   * receiver, located at `internal/service/receiver/service.go`, serves as the gateway to the system. It:
     * receives files and provides them when requested.
     * cleans up files in case of an interrupted upload.
   * orchestrator, situated in `internal/service/orchestrator/service.go`, handles data requests and storage nodes, which can be in two states: ready and not ready. It:
     * routes data requests to storage nodes.
     * manages the addition of new storage nodes and oversees the rebalancing process from old nodes to new ones.
   * storage node, based in `internal/service/storager/service.go`, is responsible for storing and serving data. It:
     * stores data and retrieves it when requested.
     * serves all sector files on demand.
2. The main test file that demonstrates how the system works can be found at `internal/service/service_test.go`
3. For simplicity, services communicate with each other directly in memory. An additional transport layer is not implemented, but it can be added as everything is based on interfaces.
4. App use consistent hashing to distribute files to storage servers, with 360 sectors and a clockwise iteration.
5. When a new storage node joins, calculations are based on other nodes' usage. See `internal/service/orchestrator/circle_test.go` for details:
   * for equal disk utilization, each node tracks its current usage. When a new node joins, it queries all current nodes about their usage and joins the node with maximum usage.
   * after a node joins, the rebalancing process starts. For simplification, it zips/unzips all sector files to the directory of the new node. Refer to `internal/service/orchestrator/service.go: AddDataKeeper`.
   * while rebalancing is in progress, the old node continues to serve requests.
   * once rebalancing is finished, all requests are redirected to the new node.

#### Improvements
* Implement dumping and restoring the storage cluster topology on server shutdown/start.
* Add a transport layer like gRPC to enable stream connections.
* Enhance the calculation of current disk usage based on real disk usage when a storage node starts.
* Implement a better transactional orchestration mechanism with retries and timeouts logic.
