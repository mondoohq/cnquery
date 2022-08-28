package internal

import (
	"fmt"
	"math"
	"sort"
	"time"

	vrs "github.com/hashicorp/go-version"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/resources"
)

type query struct {
	codeBundle         *llx.CodeBundle
	requiredProps      map[string]string
	resolvedProperties map[string]*llx.Primitive
}

type GraphBuilder struct {
	// queries is a map of QrID to query
	queries []query
	// datapointCollectors contains the collectors which will receive
	// datapoints
	datapointCollectors []DatapointCollector
	// collectDatapointChecksums specifies additional datapoints outside
	// the reporting job to collect
	collectDatapointChecksums []string
	// datapointType is a map of checksum to type for datapoint type
	// conversion. This is sourced from the compiled query
	datapointType map[string]string
	// progressReporter is a configured interface to receive progress
	// updates
	progressReporter ProgressReporter
	// mondooVersion is the version of mondoo. This is generally sourced
	// from the binary, but is configurable to make testing easier
	mondooVersion string
	// queryTimeout is the amount of time to wait for the underlying lumi
	// runtime to send all the expected datapoints.
	queryTimeout time.Duration

	useV2Code             bool
	featureBoolAssertions bool
}

func NewBuilder() *GraphBuilder {
	return &GraphBuilder{
		queries:                   []query{},
		datapointCollectors:       []DatapointCollector{},
		collectDatapointChecksums: []string{},
		datapointType:             map[string]string{},
		progressReporter:          NoopProgressReporter{},
		mondooVersion:             cnquery.GetCoreVersion(),
		queryTimeout:              5 * time.Minute,
	}
}

// AddQuery adds the provided code to be executed to the graph
func (b *GraphBuilder) AddQuery(c *llx.CodeBundle, propertyChecksums map[string]string, resolvedProperties map[string]*llx.Primitive) {
	b.queries = append(b.queries, query{
		codeBundle:         c,
		requiredProps:      propertyChecksums,
		resolvedProperties: resolvedProperties,
	})
}

func (b *GraphBuilder) AddDatapointType(datapointChecksum string, typ string) {
	b.datapointType[datapointChecksum] = typ
}

// CollectDatapoint requests the provided checksum be collected and sent to
// the configured datapoint collectors
func (b *GraphBuilder) CollectDatapoint(datapointChecksum string) {
	b.collectDatapointChecksums = append(b.collectDatapointChecksums, datapointChecksum)
}

// AddDatapointCollector adds a datapoint collector. Collected datapoints
// will be sent to all the provided datapoint collectors
func (b *GraphBuilder) AddDatapointCollector(c DatapointCollector) {
	b.datapointCollectors = append(b.datapointCollectors, c)
}

// WithProgressReporter sets the interface which will receive progress updates
func (b *GraphBuilder) WithProgressReporter(r ProgressReporter) {
	b.progressReporter = r
}

// WithMondooVersion sets the version of mondoo
func (b *GraphBuilder) WithMondooVersion(mondooVersion string) {
	b.mondooVersion = mondooVersion
}

// WithMondooVersion sets the version of mondoo
func (b *GraphBuilder) WithQueryTimeout(timeout time.Duration) {
	b.queryTimeout = timeout
}

func (b *GraphBuilder) WithUseV2Code(useV2Code bool) {
	b.useV2Code = useV2Code
}

func (b *GraphBuilder) WithFeatureBoolAssertions(featureBoolAssertions bool) {
	b.featureBoolAssertions = featureBoolAssertions
}

func (b *GraphBuilder) Build(schema *resources.Schema, runtime *resources.Runtime, assetMrn string) (*GraphExecutor, error) {
	resultChan := make(chan *llx.RawResult, 128)

	queries := make(map[string]query, len(b.queries))
	if b.useV2Code {
		for _, q := range b.queries {
			queries[q.codeBundle.GetCodeV2().GetId()] = q
		}
	} else {
		for _, q := range b.queries {
			queries[q.codeBundle.GetDeprecatedV5Code().GetId()] = q
		}
	}

	ge := &GraphExecutor{
		nodes:        map[NodeID]*Node{},
		edges:        map[NodeID][]NodeID{},
		priorityMap:  map[NodeID]int{},
		queryTimeout: b.queryTimeout,
		executionManager: newExecutionManager(schema, runtime, make(chan runQueueItem, len(queries)),
			resultChan, b.queryTimeout, b.useV2Code),
		resultChan: resultChan,
		doneChan:   make(chan struct{}),
	}

	ge.nodes[DatapointCollectorID] = &Node{
		id:       DatapointCollectorID,
		nodeType: DatapointCollectorNodeType,
		data: &DatapointCollectorNodeData{
			unreported: map[string]*llx.RawResult{},
			collectors: b.datapointCollectors,
		},
	}

	unrunnableQueries := []query{}

	var mondooVersion *vrs.Version
	if b.mondooVersion != "" && b.mondooVersion != "unstable" {
		var err error
		mondooVersion, err = vrs.NewVersion(b.mondooVersion)
		if err != nil {
			log.Warn().Err(err).Str("version", b.mondooVersion).Msg("unable to parse mondoo version")
		}
	}

	for queryID, q := range queries {
		canRun := checkVersion(q.codeBundle, mondooVersion)
		if canRun {
			ge.addExecutionQueryNode(queryID, q, q.resolvedProperties, b.datapointType)
		} else {
			unrunnableQueries = append(unrunnableQueries, q)
		}
	}

	datapointsToCollect := make([]string, len(b.collectDatapointChecksums))
	copy(datapointsToCollect, b.collectDatapointChecksums)

	for _, datapointChecksum := range datapointsToCollect {
		ge.addEdge(NodeID(datapointChecksum), DatapointCollectorID)
	}

	ge.handleUnrunnableQueries(unrunnableQueries)

	ge.createFinisherNode(b.progressReporter)

	for nodeID := range ge.nodes {
		prioritizeNode(ge.nodes, ge.edges, ge.priorityMap, nodeID)
	}

	// The finisher is the lowest priority node. This makes it so that
	// when a recalculation is triggered through a datapoint being reported,
	// the finisher only gets notified after all other intermediate nodes are
	// notified
	ge.priorityMap[CollectionFinisherID] = math.MinInt

	return ge, nil
}

// handleUnrunnableQueries takes the queries for which the running version does
// to meet the minimum version requirement and marks the datapoints as error.
// This is only done for datapoints which will not be reported by a runnable query
func (ge *GraphExecutor) handleUnrunnableQueries(unrunnableQueries []query) {
	for _, q := range unrunnableQueries {
		for _, checksum := range CodepointChecksums(q.codeBundle, ge.executionManager.useV2Code) {
			if _, ok := ge.nodes[NodeID(checksum)]; ok {
				// If the datapoint will be reported by another query, skip
				// handling it
				continue
			}

			ge.addDatapointNode(
				checksum,
				nil,
				&llx.RawResult{
					CodeID: checksum,
					Data: &llx.RawData{
						Error: fmt.Errorf("Unable to run query, mondoo client version %s required", q.codeBundle.MinMondooVersion),
					},
				})
		}
	}
}

func (ge *GraphExecutor) addEdge(from NodeID, to NodeID) {
	ge.edges[from] = insertSorted(ge.edges[from], to)
}

func (ge *GraphExecutor) createFinisherNode(r ProgressReporter) {
	nodeID := CollectionFinisherID
	nodeData := &CollectionFinisherNodeData{
		remainingDatapoints: make(map[string]struct{}, len(ge.nodes)),
		doneChan:            ge.doneChan,
		progressReporter:    r,
	}

	for datapointNodeID, n := range ge.nodes {
		if n.nodeType == DatapointNodeType {
			ge.addEdge(datapointNodeID, nodeID)
			nodeData.remainingDatapoints[datapointNodeID] = struct{}{}
		}
	}
	totalDatapoints := len(nodeData.remainingDatapoints)
	nodeData.totalDatapoints = totalDatapoints

	ge.nodes[nodeID] = &Node{
		id:       nodeID,
		nodeType: CollectionFinisherNodeType,
		data:     nodeData,
	}
}

func (ge *GraphExecutor) addExecutionQueryNode(queryID string, q query, resolvedProperties map[string]*llx.Primitive, datapointTypeMap map[string]string) {
	n, ok := ge.nodes[NodeID(queryID)]
	if ok {
		return
	}

	codeBundle := q.codeBundle

	nodeData := &ExecutionQueryNodeData{
		queryID:            queryID,
		codeBundle:         codeBundle,
		useV2Code:          ge.executionManager.useV2Code,
		requiredProperties: map[string]*executionQueryProperty{},
		runState:           notReadyQueryNotReady,
		runQueue:           ge.executionManager.runQueue,
	}

	n = &Node{
		id:       NodeID(string(ExecutionQueryNodeType) + "/" + queryID),
		nodeType: ExecutionQueryNodeType,
		data:     nodeData,
	}

	// These don't report anything, but they make the graph connected
	for _, checksum := range CodepointChecksums(codeBundle, ge.executionManager.useV2Code) {
		var expectedType *string
		if t, ok := datapointTypeMap[checksum]; ok {
			expectedType = &t
		}
		ge.addDatapointNode(checksum, expectedType, nil)
		ge.addEdge(n.id, NodeID(checksum))
	}

	for name, checksum := range q.requiredProps {
		nodeData.requiredProperties[name] = &executionQueryProperty{
			name:     name,
			checksum: checksum,
			resolved: false,
			value:    nil,
		}
		ge.addEdge(NodeID(checksum), n.id)
	}

	for name, val := range resolvedProperties {
		if rp, ok := nodeData.requiredProperties[name]; !ok {
			nodeData.requiredProperties[name] = &executionQueryProperty{
				name:     name,
				checksum: "",
				resolved: true,
				value: &llx.Result{
					Data: val,
				},
			}
		} else {
			rp.value = &llx.Result{
				Data: val,
			}
			rp.resolved = true
		}
	}

	ge.nodes[n.id] = n
}

func (ge *GraphExecutor) addDatapointNode(datapointChecksum string, expectedType *string, res *llx.RawResult) {
	n, ok := ge.nodes[NodeID(datapointChecksum)]
	if ok {
		return
	}

	nodeData := &DatapointNodeData{
		expectedType: expectedType,
		useV2Code:    ge.executionManager.useV2Code,
		isReported:   res != nil,
		res:          res,
	}
	n = &Node{
		id:       NodeID(datapointChecksum),
		nodeType: DatapointNodeType,
		data:     nodeData,
	}

	ge.nodes[NodeID(datapointChecksum)] = n
}

// prioritizeNode assigns each node in the graph a priority. The priority makes graph traversal
// act like a breadth-first search, minimizing the number of recalculations needed for each node.
// For example, the reporting job with a query id of the asset will have a lower priority than
// reporting jobs which have a query id of a policy mrn. In a similar way, the reporting jobs
// that have a query id of policy mrns have a lower priority than reporting jobs for queries.
// This means that if a batch of data arrives, all query reporting jobs will be recalculated first.
// The policy reporting jobs will be calculated after that, and then the asset reporting job.
func prioritizeNode(nodes map[NodeID]*Node, edges map[NodeID][]NodeID, priorityMap map[NodeID]int, n NodeID) int {
	if d, ok := priorityMap[n]; ok {
		return d
	}
	childrenMaxDepth := 0
	for _, v := range edges[n] {
		childDepth := prioritizeNode(nodes, edges, priorityMap, v)
		if childDepth > childrenMaxDepth {
			childrenMaxDepth = childDepth
		}
	}
	myDepth := childrenMaxDepth + 1
	priorityMap[n] = myDepth
	return myDepth
}

func checkVersion(codeBundle *llx.CodeBundle, curMin *vrs.Version) bool {
	if curMin != nil && codeBundle.MinMondooVersion != "" {
		requiredVer := codeBundle.MinMondooVersion
		reqMin, err := vrs.NewVersion(requiredVer)
		if err == nil && curMin.LessThan(reqMin) {
			return false
		}
	}
	return true
}

func insertSorted(ss []string, s string) []string {
	i := sort.SearchStrings(ss, s)
	if i < len(ss) && ss[i] == s {
		return ss
	}
	ss = append(ss, "")
	copy(ss[i+1:], ss[i:])
	ss[i] = s
	return ss
}

func CodepointChecksums(codeBundle *llx.CodeBundle, useV2Code bool) []string {
	return append(EntrypointChecksums(codeBundle, useV2Code),
		DatapointChecksums(codeBundle, useV2Code)...)
}

func EntrypointChecksums(codeBundle *llx.CodeBundle, useV2Code bool) []string {
	var checksums []string
	if useV2Code {
		// TODO (jaym): double check with dom this is the way to get entrypoints
		checksums = make([]string, len(codeBundle.CodeV2.Blocks[0].Entrypoints))
		for i, ref := range codeBundle.CodeV2.Blocks[0].Entrypoints {
			checksums[i] = codeBundle.CodeV2.Checksums[ref]
		}
	} else {
		checksums = make([]string, len(codeBundle.DeprecatedV5Code.Entrypoints))
		for i, ref := range codeBundle.DeprecatedV5Code.Entrypoints {
			checksums[i] = codeBundle.DeprecatedV5Code.Checksums[ref]
		}
	}
	return checksums
}

func DatapointChecksums(codeBundle *llx.CodeBundle, useV2Code bool) []string {
	var checksums []string
	if useV2Code {
		// TODO (jaym): double check with dom this is the way to get entrypoints
		checksums = make([]string, len(codeBundle.CodeV2.Blocks[0].Datapoints))
		for i, ref := range codeBundle.CodeV2.Blocks[0].Datapoints {
			checksums[i] = codeBundle.CodeV2.Checksums[ref]
		}
	} else {
		checksums = make([]string, len(codeBundle.DeprecatedV5Code.Datapoints))
		for i, ref := range codeBundle.DeprecatedV5Code.Datapoints {
			checksums[i] = codeBundle.DeprecatedV5Code.Checksums[ref]
		}
	}
	return checksums
}
