package d2

import (
	"bytes"
	"context"
	"embed"
	"errors"
	"fmt"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/holydocs/holydocs/pkg/holydocs"
	"oss.terrastruct.com/d2/d2graph"
	"oss.terrastruct.com/d2/d2layouts/d2elklayout"
	"oss.terrastruct.com/d2/d2lib"
	"oss.terrastruct.com/d2/d2renderers/d2svg"
	"oss.terrastruct.com/d2/lib/log"
	"oss.terrastruct.com/d2/lib/textmeasure"
)

// Errors.
var (
	ErrFormatSchemaNotSupported = errors.New("FormatSchema is not supported - use the Generate* methods instead")
	ErrUnsupportedFormatType    = errors.New("unsupported format type")
	ErrContextRequired          = errors.New("context cannot be nil")
	ErrTemplateParsing          = errors.New("failed to parse template")
	ErrDiagramCompilation       = errors.New("failed to compile diagram")
	ErrSVGRendering             = errors.New("failed to render SVG")
)

// Async operations.
const (
	asyncOpSend  = "send"
	asyncOpReply = "reply"
)

// Async labels.
const (
	asyncLabelPub    = "pub"
	asyncLabelPubReq = "pub/req"
	asyncLabelReq    = "req"
	requestsLabel    = "requests"
)

// D2 diagram settings.
const (
	d2PadSize = 64
)

// Text formatting.
const (
	maxWordsPerLine = 7
	wordsPerLine    = 7
	wordsOffset     = 6
)

// Map capacity multipliers.
const (
	mapCapacityMultiplier = 2
)

// Timeouts.
const (
	defaultRenderTimeout = 30 * time.Second
)

//go:embed templates/*.tmpl
var templatesFS embed.FS

const targetType = holydocs.TargetType("d2")

// Target implements the holydocs.Target interface for D2 diagrams.
type Target struct {
	overviewTemplate             *template.Template
	serviceRelationshipsTemplate *template.Template
	systemTemplate               *template.Template
	renderOpts                   *d2svg.RenderOpts
}

// NewTarget creates a new D2 target with default settings.
func NewTarget() (*Target, error) {
	overviewTemplate, err := template.ParseFS(templatesFS, "templates/overview.tmpl")
	if err != nil {
		return nil, fmt.Errorf("%w %q: %w", ErrTemplateParsing, "templates/overview.tmpl", err)
	}

	serviceRelationshipsTemplate, err := template.ParseFS(templatesFS, "templates/service_relationships.tmpl")
	if err != nil {
		return nil, fmt.Errorf("%w %q: %w", ErrTemplateParsing, "templates/service_relationships.tmpl", err)
	}

	systemTemplate, err := template.ParseFS(templatesFS, "templates/system.tmpl")
	if err != nil {
		return nil, fmt.Errorf("%w %q: %w", ErrTemplateParsing, "templates/system.tmpl", err)
	}

	return &Target{
		overviewTemplate:             overviewTemplate,
		serviceRelationshipsTemplate: serviceRelationshipsTemplate,
		systemTemplate:               systemTemplate,
		renderOpts: &d2svg.RenderOpts{
			Pad: func() *int64 {
				p := int64(d2PadSize)

				return &p
			}(),
		},
	}, nil
}

// Capabilities returns the capabilities of the D2 target.
func (t *Target) Capabilities() holydocs.TargetCapabilities {
	return holydocs.TargetCapabilities{
		Format: true,
		Render: true,
	}
}

// FormatSchema formats a schema according to the specified options.
// This method is required by the SchemaFormatter interface but is not used during documentation generation.
func (t *Target) FormatSchema(
	_ context.Context,
	_ holydocs.Schema,
	_ holydocs.FormatOptions,
) (holydocs.FormattedSchema, error) {
	return holydocs.FormattedSchema{}, ErrFormatSchemaNotSupported
}

// RenderSchema renders a formatted schema to SVG.
func (t *Target) RenderSchema(ctx context.Context, fs holydocs.FormattedSchema) ([]byte, error) {
	if ctx == nil {
		return nil, ErrContextRequired
	}

	if fs.Type != targetType {
		return nil, fmt.Errorf("%w: %s, expected: %s", ErrUnsupportedFormatType, fs.Type, targetType)
	}

	// Add timeout handling
	ctx, cancel := context.WithTimeout(ctx, defaultRenderTimeout)
	defer cancel()

	ctx = log.WithDefault(ctx)

	// Create a new Ruler for each call since it's not thread-safe
	ruler, err := textmeasure.NewRuler()
	if err != nil {
		return nil, fmt.Errorf("creating ruler: %w", err)
	}

	layoutResolver := func(_ string) (d2graph.LayoutGraph, error) {
		return d2elklayout.DefaultLayout, nil
	}

	compileOpts := &d2lib.CompileOptions{
		LayoutResolver: layoutResolver,
		Ruler:          ruler,
	}

	diagram, _, err := d2lib.Compile(ctx, string(fs.Data), compileOpts, t.renderOpts)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDiagramCompilation, err)
	}

	svg, err := d2svg.Render(diagram, t.renderOpts)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrSVGRendering, err)
	}

	return svg, nil
}

type AsyncEdge struct {
	Source  string
	Target  string
	Channel string
	Kind    string
}

// ServiceMaps contains service-related maps for efficient lookups.
type ServiceMaps struct {
	ServiceNames map[string]struct{}
	ServiceIDs   map[string]string
}

// ServiceRelationshipEdges contains edges and async edges for a service.
type ServiceRelationshipEdges struct {
	Edges      []diagramEdgeDocs
	AsyncEdges []AsyncEdge
}

func serviceNodeID(name string) string {
	return "service_" + sanitizeFilename(name)
}

func externalNodeID(name string) string {
	return "external_" + sanitizeFilename(name)
}

func systemNodeID(name string) string {
	return "system_" + sanitizeFilename(name)
}

func sanitizeFilename(name string) string {
	return strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(name, " ", "-"), "_", "-"))
}

func formatOverviewDescription(description string) string {
	if description == "" {
		return ""
	}

	words := strings.Fields(description)
	if len(words) <= maxWordsPerLine {
		return description
	}

	lines := make([]string, 0, (len(words)+wordsOffset)/wordsPerLine)
	for i := 0; i < len(words); i += wordsPerLine {
		end := i + wordsPerLine
		if end > len(words) {
			end = len(words)
		}
		lines = append(lines, strings.Join(words[i:end], " "))
	}

	return strings.Join(lines, "  \n")
}

func FormatDescription(description string) string {
	if description == "" {
		return ""
	}

	words := strings.Fields(description)
	if len(words) <= maxWordsPerLine {
		return description
	}

	lines := make([]string, 0, (len(words)+wordsOffset)/wordsPerLine)
	for i := 0; i < len(words); i += wordsPerLine {
		end := i + wordsPerLine
		if end > len(words) {
			end = len(words)
		}
		lines = append(lines, strings.Join(words[i:end], " "))
	}

	return strings.Join(lines, " ")
}

func shapeForTechnologies(techs map[string]string) string {
	if len(techs) == 0 {
		return "rectangle"
	}

	lowered := make(map[string]struct{}, len(techs)*mapCapacityMultiplier)
	replacer := strings.NewReplacer(" ", "", "-", "", "_", "")
	for key := range techs {
		lowered[key] = struct{}{}
		compact := replacer.Replace(key)
		lowered[compact] = struct{}{}
	}

	matchLower := func(options ...string) bool {
		for _, opt := range options {
			if _, exists := lowered[opt]; exists {
				return true
			}
		}

		return false
	}

	if matchLower("kafka", "rabbitmq", "nats", "sqs", "pubsub") {
		return "queue"
	}

	if matchLower(
		"postgres", "postgresql", "mysql", "mongodb", "redis",
		"cassandra", "elasticsearch", "dynamodb", "sqlite",
		"clickhouse", "aurora", "mssql", "sqlserver", "oracle", "snowflake",
	) {
		return "cylinder"
	}

	return "rectangle"
}

type edgeSummary struct {
	outSend  map[string]struct{}
	outReply map[string]struct{}
	inSend   map[string]struct{}
	inReply  map[string]struct{}
}

func AggregateAsyncEdgesForService(serviceName string, asyncEdges []AsyncEdge,
	serviceNames map[string]struct{}) ([]DiagramEdge, []AsyncSummary) {
	summaries := buildEdgeSummaries(serviceName, asyncEdges, serviceNames)

	return buildDiagramEdgesAndSummaries(serviceName, summaries)
}

func buildEdgeSummaries(serviceName string, asyncEdges []AsyncEdge,
	serviceNames map[string]struct{}) map[string]*edgeSummary {
	summaries := make(map[string]*edgeSummary)

	for _, edge := range asyncEdges {
		// Skip edges not involving this service
		if edge.Source != serviceName && edge.Target != serviceName {
			continue
		}

		other := edge.Source
		if other == serviceName {
			other = edge.Target
		}

		// Skip if other is not a service
		if _, isService := serviceNames[other]; !isService {
			continue
		}

		summary, exists := summaries[other]
		if !exists {
			summary = &edgeSummary{
				outSend:  make(map[string]struct{}),
				outReply: make(map[string]struct{}),
				inSend:   make(map[string]struct{}),
				inReply:  make(map[string]struct{}),
			}
			summaries[other] = summary
		}

		updateEdgeSummary(summary, edge, serviceName)
	}

	return summaries
}

func updateEdgeSummary(summary *edgeSummary, edge AsyncEdge, serviceName string) {
	if edge.Source == serviceName {
		// Outgoing edge
		switch edge.Kind {
		case asyncOpSend:
			summary.outSend[edge.Channel] = struct{}{}
		case asyncOpReply:
			summary.outReply[edge.Channel] = struct{}{}
		}
	} else {
		// Incoming edge
		switch edge.Kind {
		case asyncOpSend:
			summary.inSend[edge.Channel] = struct{}{}
		case asyncOpReply:
			summary.inReply[edge.Channel] = struct{}{}
		}
	}
}

func buildDiagramEdgesAndSummaries(serviceName string,
	summaries map[string]*edgeSummary) ([]DiagramEdge, []AsyncSummary) {
	mainID := serviceNodeID(serviceName)
	diagramEdges := make([]DiagramEdge, 0, len(summaries)*mapCapacityMultiplier)
	textSummaries := make([]AsyncSummary, 0, len(summaries)*mapCapacityMultiplier)

	for other, summary := range summaries {
		otherID := serviceNodeID(other)

		if len(summary.outSend) > 0 {
			if label := deriveAsyncLabel(summary.outSend, summary.outReply); label != "" {
				diagramEdges = append(diagramEdges, DiagramEdge{From: mainID, To: otherID, Label: label})
				textSummaries = append(textSummaries, AsyncSummary{
					Direction: describeAsyncDirection(label, true),
					Target:    other,
					Label:     label,
				})
			}
		}

		if len(summary.inSend) > 0 {
			if label := deriveAsyncLabel(summary.inSend, summary.inReply); label != "" {
				diagramEdges = append(diagramEdges, DiagramEdge{From: otherID, To: mainID, Label: label})
				textSummaries = append(textSummaries, AsyncSummary{
					Direction: describeAsyncDirection(label, false),
					Target:    other,
					Label:     label,
				})
			}
		}
	}

	return diagramEdges, textSummaries
}

type DiagramEdge struct {
	From  string
	To    string
	Label string
}

type AsyncSummary struct {
	Direction string
	Target    string
	Label     string
}

func deriveAsyncLabel(sendChannels, replyChannels map[string]struct{}) string {
	if len(sendChannels) == 0 {
		return ""
	}
	if len(replyChannels) == 0 {
		return asyncLabelPub
	}
	if hasNonReplyChannels(sendChannels, replyChannels) {
		return asyncLabelPubReq
	}

	return asyncLabelReq
}

func hasNonReplyChannels(sendChannels, replyChannels map[string]struct{}) bool {
	for channel := range sendChannels {
		if _, isReply := replyChannels[channel]; !isReply {
			return true
		}
	}

	return false
}

func describeAsyncDirection(label string, isOutgoing bool) string {
	if isOutgoing {
		switch label {
		case "pub":
			return "publishes to"
		case "req":
			return "requests to"
		case "pub/req":
			return "publishes to and requests from"
		}
	} else {
		switch label {
		case "pub":
			return "receives from"
		case "req":
			return "handles requests from"
		case "pub/req":
			return "receives from and replies to"
		}
	}

	return label
}

// Docs-specific diagram generation

// OverviewDocsNode represents a node in the overview diagram for docs generation.
type OverviewDocsNode struct {
	ID       string
	Label    string
	IsSystem bool
	External bool
	Internal bool
	Person   bool
	Content  string
}

// OverviewDocsEdge represents an edge in the overview diagram for docs generation.
type OverviewDocsEdge struct {
	From  string
	To    string
	Label string
}

// OverviewDocsPayload represents the data structure for overview docs template.
type OverviewDocsPayload struct {
	Nodes               []OverviewDocsNode
	Edges               []OverviewDocsEdge
	HasInternalServices bool
	GlobalName          string
}

// ServiceRelationshipsDocsNode represents a service node in service relationships diagram.
type ServiceRelationshipsDocsNode struct {
	ID    string
	Label string
}

// ServiceRelationshipsDocsExternalNode represents an external node in service relationships diagram.
type ServiceRelationshipsDocsExternalNode struct {
	ID       string
	Label    string
	Shape    string
	Tooltip  string
	External bool
	Person   bool
}

// ServiceRelationshipsDocsEdge represents an edge in service relationships diagram.
type ServiceRelationshipsDocsEdge struct {
	From  string
	To    string
	Label string
}

// ServiceRelationshipsDocsPayload represents the data structure for service relationships docs template.
type ServiceRelationshipsDocsPayload struct {
	Services      []ServiceRelationshipsDocsNode
	ExternalNodes []ServiceRelationshipsDocsExternalNode
	Edges         []ServiceRelationshipsDocsEdge
}

// SystemDocsNode represents a service node in system diagram for docs generation.
type SystemDocsNode struct {
	ID       string
	Label    string
	Content  string
	External bool
	Person   bool
}

// SystemDocsEdge represents an edge in system diagram for docs generation.
type SystemDocsEdge struct {
	From  string
	To    string
	Label string
}

// SystemDocsPayload represents the data structure for system docs template.
type SystemDocsPayload struct {
	SystemName    string
	SystemID      string
	SystemNodes   []SystemDocsNode
	ExternalNodes []SystemDocsNode
	Edges         []SystemDocsEdge
}

// GenerateOverviewDiagram generates an overview diagram using the docs-specific template.
func (t *Target) GenerateOverviewDiagram(ctx context.Context, schema holydocs.Schema,
	asyncEdges []AsyncEdge, globalName string) ([]byte, error) {
	payload := t.prepareOverviewDocsPayload(schema, asyncEdges, globalName)

	var buf bytes.Buffer
	if err := t.overviewTemplate.Execute(&buf, payload); err != nil {
		return nil, fmt.Errorf("execute overview docs template: %w", err)
	}

	formatted := holydocs.FormattedSchema{
		Type: targetType,
		Data: buf.Bytes(),
	}

	return t.RenderSchema(ctx, formatted)
}

// GenerateServiceRelationshipsDiagram generates a service relationships diagram using the docs-specific template.
func (t *Target) GenerateServiceRelationshipsDiagram(ctx context.Context, service holydocs.Service,
	allServices []holydocs.Service, asyncEdges []AsyncEdge) ([]byte, error) {
	payload := t.prepareServiceRelationshipsDocsPayload(service, allServices, asyncEdges)

	var buf bytes.Buffer
	if err := t.serviceRelationshipsTemplate.Execute(&buf, payload); err != nil {
		return nil, fmt.Errorf("execute service relationships docs template: %w", err)
	}

	formatted := holydocs.FormattedSchema{
		Type: targetType,
		Data: buf.Bytes(),
	}

	return t.RenderSchema(ctx, formatted)
}

// GenerateOverviewDiagramScript generates the D2 script for overview diagram.
func (t *Target) GenerateOverviewDiagramScript(schema holydocs.Schema, asyncEdges []AsyncEdge,
	globalName string) ([]byte, error) {
	payload := t.prepareOverviewDocsPayload(schema, asyncEdges, globalName)

	var buf bytes.Buffer
	if err := t.overviewTemplate.Execute(&buf, payload); err != nil {
		return nil, fmt.Errorf("execute overview docs template: %w", err)
	}

	return buf.Bytes(), nil
}

// GenerateSystemDiagram generates a system diagram using the docs-specific template.
func (t *Target) GenerateSystemDiagram(ctx context.Context, schema holydocs.Schema,
	systemName string, asyncEdges []AsyncEdge) ([]byte, error) {
	payload := t.prepareSystemDocsPayload(schema, systemName, asyncEdges)

	var buf bytes.Buffer
	if err := t.systemTemplate.Execute(&buf, payload); err != nil {
		return nil, fmt.Errorf("execute system docs template: %w", err)
	}

	formatted := holydocs.FormattedSchema{
		Type: targetType,
		Data: buf.Bytes(),
	}

	return t.RenderSchema(ctx, formatted)
}

// GenerateSystemDiagramScript generates the D2 script for system diagram.
func (t *Target) GenerateSystemDiagramScript(schema holydocs.Schema, systemName string,
	asyncEdges []AsyncEdge) ([]byte, error) {
	payload := t.prepareSystemDocsPayload(schema, systemName, asyncEdges)

	var buf bytes.Buffer
	if err := t.systemTemplate.Execute(&buf, payload); err != nil {
		return nil, fmt.Errorf("execute system docs template: %w", err)
	}

	return buf.Bytes(), nil
}

// GenerateServiceRelationshipsDiagramScript generates the D2 script for service relationships diagram.
func (t *Target) GenerateServiceRelationshipsDiagramScript(service holydocs.Service,
	allServices []holydocs.Service, asyncEdges []AsyncEdge) ([]byte, error) {
	payload := t.prepareServiceRelationshipsDocsPayload(service, allServices, asyncEdges)

	var buf bytes.Buffer
	if err := t.serviceRelationshipsTemplate.Execute(&buf, payload); err != nil {
		return nil, fmt.Errorf("execute service relationships docs template: %w", err)
	}

	return buf.Bytes(), nil
}

func (t *Target) prepareOverviewDocsPayload(schema holydocs.Schema, asyncEdges []AsyncEdge,
	globalName string) OverviewDocsPayload {
	payload := OverviewDocsPayload{
		Nodes: []OverviewDocsNode{},
		Edges: []OverviewDocsEdge{},
	}

	serviceToNode, nodes, idToServiceName := buildOverviewNodes(schema)
	if len(nodes) == 0 {
		return payload
	}

	edgeSet := make(map[string]OverviewDocsEdge)
	edgesByService := buildEdgesByServiceMap(asyncEdges)

	processOverviewRelationships(schema, serviceToNode, nodes, edgeSet)
	processOverviewAsyncEdges(schema, edgesByService, serviceToNode, idToServiceName, edgeSet, t)

	buildOverviewPayload(&payload, nodes, edgeSet, globalName)

	return payload
}

func (t *Target) prepareServiceRelationshipsDocsPayload(service holydocs.Service, allServices []holydocs.Service,
	asyncEdges []AsyncEdge) ServiceRelationshipsDocsPayload {
	payload := ServiceRelationshipsDocsPayload{
		Services:      []ServiceRelationshipsDocsNode{},
		ExternalNodes: []ServiceRelationshipsDocsExternalNode{},
		Edges:         []ServiceRelationshipsDocsEdge{},
	}

	serviceMaps := buildServiceMaps(allServices)
	externalNodes := make(map[string]*externalNodeDocs)
	serviceEdges := buildServiceRelationshipEdges(service, serviceMaps.ServiceNames, externalNodes, asyncEdges, t)

	definedNodes := make(map[string]struct{})
	defineServiceNode := createServiceNodeDefiner(&payload, definedNodes)

	defineAllServiceNodes(service, serviceEdges.AsyncEdges, serviceEdges.Edges,
		serviceMaps.ServiceNames, serviceMaps.ServiceIDs, defineServiceNode)
	addExternalNodesToPayload(&payload, externalNodes)
	sortAndConvertEdges(&payload, serviceEdges.Edges)

	return payload
}

func buildServiceMaps(allServices []holydocs.Service) ServiceMaps {
	serviceNames := make(map[string]struct{}, len(allServices))
	serviceIDs := make(map[string]string, len(allServices))
	for _, svc := range allServices {
		serviceNames[svc.Info.Name] = struct{}{}
		serviceIDs[serviceNodeID(svc.Info.Name)] = svc.Info.Name
	}

	return ServiceMaps{
		ServiceNames: serviceNames,
		ServiceIDs:   serviceIDs,
	}
}

func buildServiceRelationshipEdges(service holydocs.Service, serviceNames map[string]struct{},
	externalNodes map[string]*externalNodeDocs, asyncEdges []AsyncEdge, t *Target) ServiceRelationshipEdges {
	filteredServices := []holydocs.Service{service}
	edges := buildRelationshipEdgesDocs(filteredServices, serviceNames, externalNodes)

	serviceOnlyEdges := filterAsyncEdgesForService(service.Info.Name, asyncEdges)
	diagEdges, _ := t.AggregateAsyncEdgesForService(service.Info.Name, serviceOnlyEdges, serviceNames)

	// Convert DiagramEdge to diagramEdgeDocs
	for _, de := range diagEdges {
		edges = append(edges, diagramEdgeDocs(de))
	}

	return ServiceRelationshipEdges{
		Edges:      edges,
		AsyncEdges: serviceOnlyEdges,
	}
}

func filterAsyncEdgesForService(serviceName string, asyncEdges []AsyncEdge) []AsyncEdge {
	serviceOnlyEdges := make([]AsyncEdge, 0, len(asyncEdges))
	for _, edge := range asyncEdges {
		if edge.Source != serviceName && edge.Target != serviceName {
			continue
		}
		serviceOnlyEdges = append(serviceOnlyEdges, edge)
	}

	return serviceOnlyEdges
}

func createServiceNodeDefiner(payload *ServiceRelationshipsDocsPayload, definedNodes map[string]struct{}) func(string) {
	return func(name string) {
		id := serviceNodeID(name)
		if _, exists := definedNodes[id]; exists {
			return
		}
		definedNodes[id] = struct{}{}
		payload.Services = append(payload.Services, ServiceRelationshipsDocsNode{
			ID:    id,
			Label: name,
		})
	}
}

func defineAllServiceNodes(service holydocs.Service, serviceOnlyEdges []AsyncEdge, edges []diagramEdgeDocs,
	serviceNames map[string]struct{}, serviceIDs map[string]string, defineServiceNode func(string)) {
	defineServiceNode(service.Info.Name)

	for _, edge := range serviceOnlyEdges {
		if _, exists := serviceNames[edge.Source]; exists {
			defineServiceNode(edge.Source)
		}
		if _, exists := serviceNames[edge.Target]; exists {
			defineServiceNode(edge.Target)
		}
	}

	for _, edge := range edges {
		if name, ok := serviceIDs[edge.From]; ok {
			defineServiceNode(name)
		}
		if name, ok := serviceIDs[edge.To]; ok {
			defineServiceNode(name)
		}
	}
}

func addExternalNodesToPayload(payload *ServiceRelationshipsDocsPayload, externalNodes map[string]*externalNodeDocs) {
	sortedExternal := make([]*externalNodeDocs, 0, len(externalNodes))
	for _, node := range externalNodes {
		sortedExternal = append(sortedExternal, node)
	}
	sort.SliceStable(sortedExternal, func(i, j int) bool {
		return sortedExternal[i].name < sortedExternal[j].name
	})

	for _, node := range sortedExternal {
		label := buildExternalNodeLabel(node)
		tooltip := buildExternalNodeTooltip(node)

		payload.ExternalNodes = append(payload.ExternalNodes, ServiceRelationshipsDocsExternalNode{
			ID:       node.id,
			Label:    label,
			Shape:    shapeForTechnologies(node.technologies),
			Tooltip:  tooltip,
			External: node.external,
			Person:   node.person,
		})
	}
}

func buildExternalNodeLabel(node *externalNodeDocs) string {
	label := node.name
	if len(node.technologies) > 0 {
		techs := make([]string, 0, len(node.technologies))
		for _, tech := range node.technologies {
			techs = append(techs, tech)
		}
		sort.Strings(techs)
		label = fmt.Sprintf("%s\\n[%s]", label, strings.Join(techs, ", "))
	}

	return label
}

func buildExternalNodeTooltip(node *externalNodeDocs) string {
	if len(node.descriptions) == 0 {
		return ""
	}

	desc := make([]string, 0, len(node.descriptions))
	for d := range node.descriptions {
		desc = append(desc, d)
	}
	sort.Strings(desc)

	return strings.Join(desc, "\n")
}

func sortAndConvertEdges(payload *ServiceRelationshipsDocsPayload, edges []diagramEdgeDocs) {
	sort.SliceStable(edges, func(i, j int) bool {
		if edges[i].From != edges[j].From {
			return edges[i].From < edges[j].From
		}
		if edges[i].To != edges[j].To {
			return edges[i].To < edges[j].To
		}

		return edges[i].Label < edges[j].Label
	})

	// Convert diagramEdgeDocs to ServiceRelationshipsDocsEdge
	payload.Edges = make([]ServiceRelationshipsDocsEdge, len(edges))
	for i, edge := range edges {
		payload.Edges[i] = ServiceRelationshipsDocsEdge(edge)
	}
}

func (t *Target) prepareSystemDocsPayload(schema holydocs.Schema, systemName string,
	asyncEdges []AsyncEdge) SystemDocsPayload {
	payload := SystemDocsPayload{
		SystemName:    systemName,
		SystemID:      sanitizeFilename(systemName),
		SystemNodes:   []SystemDocsNode{},
		ExternalNodes: []SystemDocsNode{},
		Edges:         []SystemDocsEdge{},
	}

	systemServices := findSystemServices(schema, systemName)
	if len(systemServices) == 0 {
		return payload
	}

	serviceToNode, nodes, idToServiceName := buildSystemNodes(systemServices)

	edgeSet := make(map[string]SystemDocsEdge)

	allowedActions := map[holydocs.RelationshipAction]struct{}{
		holydocs.RelationshipActionRequests: {},
		holydocs.RelationshipActionReplies:  {},
		holydocs.RelationshipActionSends:    {},
		holydocs.RelationshipActionReceives: {},
	}

	processSystemRelationships(schema, systemServices, serviceToNode, nodes, edgeSet)

	processExternalServiceRelationships(schema, systemServices, serviceToNode, nodes, edgeSet, allowedActions)

	processPersonRelationships(schema, systemServices, serviceToNode, nodes, edgeSet, allowedActions)

	edgesByService := processSystemAsyncEdges(systemServices, asyncEdges, serviceToNode, idToServiceName, edgeSet, t)

	processExternalAsyncEdges(schema, systemServices, serviceToNode, nodes, edgeSet, edgesByService)

	buildSystemPayload(&payload, nodes, edgeSet, systemServices)

	return payload
}

// Helper types and functions for docs generation

type externalNodeDocs struct {
	id           string
	name         string
	technologies map[string]string
	descriptions map[string]struct{}
	external     bool
	person       bool
}

type diagramEdgeDocs struct {
	From  string
	To    string
	Label string
}

func buildRelationshipEdgesDocs(
	services []holydocs.Service,
	serviceNames map[string]struct{},
	externalNodes map[string]*externalNodeDocs,
) []diagramEdgeDocs {
	edgeSet := make(map[string]diagramEdgeDocs)

	for _, service := range services {
		processServiceRelationships(service, serviceNames, externalNodes, edgeSet)
	}

	edges := make([]diagramEdgeDocs, 0, len(edgeSet))
	for _, edge := range edgeSet {
		edges = append(edges, edge)
	}

	return edges
}

func processServiceRelationships(
	service holydocs.Service,
	serviceNames map[string]struct{},
	externalNodes map[string]*externalNodeDocs,
	edgeSet map[string]diagramEdgeDocs,
) {
	serviceID := serviceNodeID(service.Info.Name)

	for _, rel := range service.Relationships {
		targetName := strings.TrimSpace(rel.Participant)
		if targetName == "" {
			continue
		}

		label := string(rel.Action)
		if rel.Action == holydocs.RelationshipActionReplies {
			label = requestsLabel
		}

		if _, exists := serviceNames[targetName]; exists {
			processServiceToServiceEdge(serviceID, targetName, label, rel, edgeSet)

			continue
		}

		processServiceToExternalEdge(serviceID, targetName, rel, externalNodes, edgeSet, label)
	}
}

func processServiceToServiceEdge(
	serviceID, targetName, label string,
	rel holydocs.Relationship,
	edgeSet map[string]diagramEdgeDocs,
) {
	targetID := serviceNodeID(targetName)
	from, to := orientedEdge(serviceID, targetID, rel.Action)
	key := fmt.Sprintf("%s|%s|%s", from, to, label)
	edgeSet[key] = diagramEdgeDocs{From: from, To: to, Label: label}
}

func processServiceToExternalEdge(
	serviceID, targetName string,
	rel holydocs.Relationship,
	externalNodes map[string]*externalNodeDocs,
	edgeSet map[string]diagramEdgeDocs,
	label string,
) {
	nodeID := externalNodeID(targetName)
	node, exists := externalNodes[nodeID]
	if !exists {
		node = &externalNodeDocs{
			id:           nodeID,
			name:         targetName,
			technologies: make(map[string]string),
			descriptions: make(map[string]struct{}),
		}
		externalNodes[nodeID] = node
	}

	updateExternalNode(node, rel)

	from, to := orientedEdge(serviceID, nodeID, rel.Action)
	key := fmt.Sprintf("%s|%s|%s", from, to, label)
	edgeSet[key] = diagramEdgeDocs{From: from, To: to, Label: label}
}

func updateExternalNode(node *externalNodeDocs, rel holydocs.Relationship) {
	if rel.Technology != "" {
		normalized := strings.ToLower(rel.Technology)
		node.technologies[normalized] = rel.Technology
	}
	if rel.Description != "" {
		node.descriptions[rel.Description] = struct{}{}
	}
	if rel.External {
		node.external = true
	}
	if rel.Person {
		node.person = true
	}
}

// AggregateAsyncEdgesForService aggregates async edges for a service and returns diagram edges and summaries.
func (t *Target) AggregateAsyncEdgesForService(serviceName string, asyncEdges []AsyncEdge,
	serviceNames map[string]struct{}) ([]DiagramEdge, []AsyncSummary) {
	summaries := make(map[string]*edgeSummary)

	ensureSummary := func(other string) *edgeSummary {
		summary, exists := summaries[other]
		if !exists {
			summary = &edgeSummary{
				outSend:  make(map[string]struct{}),
				outReply: make(map[string]struct{}),
				inSend:   make(map[string]struct{}),
				inReply:  make(map[string]struct{}),
			}
			summaries[other] = summary
		}

		return summary
	}

	for _, edge := range asyncEdges {
		processEdgeForService(edge, serviceName, serviceNames, summaries, ensureSummary)
	}

	mainID := serviceNodeID(serviceName)
	diagramEdges := make([]DiagramEdge, 0, len(summaries)*mapCapacityMultiplier)
	textSummaries := make([]AsyncSummary, 0, len(summaries)*mapCapacityMultiplier)

	for other, summary := range summaries {
		otherID := serviceNodeID(other)

		if len(summary.outSend) > 0 {
			if label := deriveAsyncLabel(summary.outSend, summary.outReply); label != "" {
				diagramEdges = append(diagramEdges, DiagramEdge{From: mainID, To: otherID, Label: label})
				textSummaries = append(textSummaries, AsyncSummary{
					Direction: describeAsyncDirection(label, true),
					Target:    other,
					Label:     label,
				})
			}
		}

		if len(summary.inSend) > 0 {
			if label := deriveAsyncLabel(summary.inSend, summary.inReply); label != "" {
				diagramEdges = append(diagramEdges, DiagramEdge{From: otherID, To: mainID, Label: label})
				textSummaries = append(textSummaries, AsyncSummary{
					Direction: describeAsyncDirection(label, false),
					Target:    other,
					Label:     label,
				})
			}
		}
	}

	return diagramEdges, textSummaries
}

func processEdgeForService(edge AsyncEdge, serviceName string, serviceNames map[string]struct{},
	_ map[string]*edgeSummary, ensureSummary func(other string) *edgeSummary) {
	switch {
	case edge.Source == serviceName:
		processOutgoingEdge(edge, serviceName, serviceNames, ensureSummary)
	case edge.Target == serviceName:
		processIncomingEdge(edge, serviceName, serviceNames, ensureSummary)
	}
}

func processOutgoingEdge(edge AsyncEdge, serviceName string, serviceNames map[string]struct{},
	ensureSummary func(other string) *edgeSummary) {
	other := edge.Target
	if other == serviceName {
		return
	}
	if _, ok := serviceNames[other]; !ok {
		return
	}

	summary := ensureSummary(other)
	switch edge.Kind {
	case asyncOpSend:
		summary.outSend[edge.Channel] = struct{}{}
	case asyncOpReply:
		summary.inReply[edge.Channel] = struct{}{}
	}
}

func processIncomingEdge(edge AsyncEdge, serviceName string, serviceNames map[string]struct{},
	ensureSummary func(other string) *edgeSummary) {
	other := edge.Source
	if other == serviceName {
		return
	}
	if _, ok := serviceNames[other]; !ok {
		return
	}

	summary := ensureSummary(other)
	switch edge.Kind {
	case asyncOpSend:
		summary.inSend[edge.Channel] = struct{}{}
	case asyncOpReply:
		summary.outReply[edge.Channel] = struct{}{}
	}
}

func buildOverviewNodes(schema holydocs.Schema) (map[string]OverviewDocsNode,
	map[string]OverviewDocsNode, map[string]string) {
	serviceToNode := make(map[string]OverviewDocsNode)
	nodes := make(map[string]OverviewDocsNode)
	idToServiceName := make(map[string]string, len(schema.Services))

	for _, service := range schema.Services {
		idToServiceName[serviceNodeID(service.Info.Name)] = service.Info.Name
	}

	for _, service := range schema.Services {
		systemName := strings.TrimSpace(service.Info.System)
		if systemName != "" {
			nodeID := systemNodeID(systemName)
			node, exists := nodes[nodeID]
			if !exists {
				node = OverviewDocsNode{
					ID:       nodeID,
					Label:    systemName,
					Internal: true, // Systems are internal
				}
				nodes[nodeID] = node
			}
			serviceToNode[service.Info.Name] = node

			continue
		}

		nodeID := serviceNodeID(service.Info.Name)
		content := buildOverviewNodeContent(service.Info)
		node := OverviewDocsNode{
			ID:       nodeID,
			Label:    service.Info.Name,
			Content:  content,
			Internal: true,
		}
		nodes[nodeID] = node
		serviceToNode[service.Info.Name] = node
	}

	return serviceToNode, nodes, idToServiceName
}

func buildEdgesByServiceMap(asyncEdges []AsyncEdge) map[string][]AsyncEdge {
	edgesByService := make(map[string][]AsyncEdge)
	for _, edge := range asyncEdges {
		edgesByService[edge.Source] = append(edgesByService[edge.Source], edge)
		edgesByService[edge.Target] = append(edgesByService[edge.Target], edge)
	}

	return edgesByService
}

func processOverviewRelationships(schema holydocs.Schema, serviceToNode map[string]OverviewDocsNode,
	nodes map[string]OverviewDocsNode, edgeSet map[string]OverviewDocsEdge) {
	allowedActions := map[holydocs.RelationshipAction]struct{}{
		holydocs.RelationshipActionRequests: {},
		holydocs.RelationshipActionReplies:  {},
		holydocs.RelationshipActionSends:    {},
		holydocs.RelationshipActionReceives: {},
	}

	for _, service := range schema.Services {
		srcNode, ok := serviceToNode[service.Info.Name]
		if !ok {
			continue
		}

		for _, rel := range service.Relationships {
			if _, allowed := allowedActions[rel.Action]; !allowed {
				continue
			}

			tgtNode := getOrCreateTargetNode(rel, serviceToNode, nodes)
			processOverviewEdge(srcNode, tgtNode, rel, edgeSet)
		}
	}
}

func getOrCreateTargetNode(rel holydocs.Relationship, serviceToNode map[string]OverviewDocsNode,
	nodes map[string]OverviewDocsNode) OverviewDocsNode {
	tgtNode, ok := serviceToNode[rel.Participant]
	if !ok {
		nodeID := externalNodeID(rel.Participant)
		node, exists := nodes[nodeID]
		if !exists {
			node = OverviewDocsNode{
				ID:       nodeID,
				Label:    rel.Participant,
				External: true,       // Both persons and external services are external
				Person:   rel.Person, // Mark as person if it's a person
				Content:  formatOverviewDescription(rel.Description),
			}
			nodes[nodeID] = node
		}
		tgtNode = node
	}

	return tgtNode
}

func processOverviewEdge(srcNode, tgtNode OverviewDocsNode, rel holydocs.Relationship,
	edgeSet map[string]OverviewDocsEdge) {
	from, to := orientedEdge(srcNode.ID, tgtNode.ID, rel.Action)
	if from == to {
		return
	}

	// Adjust node IDs for internal services
	// After edge reversal, we need to check which node is actually the source/target
	fromID := from
	toID := to

	// Determine which node is the actual source and target after reversal
	var actualSrcNode, actualTgtNode OverviewDocsNode
	if from == srcNode.ID {
		actualSrcNode = srcNode
		actualTgtNode = tgtNode
	} else {
		actualSrcNode = tgtNode
		actualTgtNode = srcNode
	}

	if actualSrcNode.Internal {
		fromID = "internal." + from
	}
	if actualTgtNode.Internal {
		toID = "internal." + to
	}

	label := string(rel.Action)
	if rel.Action == holydocs.RelationshipActionReplies {
		label = requestsLabel
	}
	key := fmt.Sprintf("%s|%s|%s|rel", fromID, toID, label)
	edgeSet[key] = OverviewDocsEdge{From: fromID, To: toID, Label: label}
}

func processOverviewAsyncEdges(schema holydocs.Schema, edgesByService map[string][]AsyncEdge,
	serviceToNode map[string]OverviewDocsNode, idToServiceName map[string]string,
	edgeSet map[string]OverviewDocsEdge, t *Target) {
	serviceNames := make(map[string]struct{}, len(schema.Services))
	for _, svc := range schema.Services {
		serviceNames[svc.Info.Name] = struct{}{}
	}

	for _, service := range schema.Services {
		diagEdges, _ := t.AggregateAsyncEdgesForService(service.Info.Name, edgesByService[service.Info.Name], serviceNames)
		for _, de := range diagEdges {
			processAsyncEdge(de, serviceToNode, idToServiceName, edgeSet)
		}
	}
}

func processAsyncEdge(de DiagramEdge, serviceToNode map[string]OverviewDocsNode,
	idToServiceName map[string]string, edgeSet map[string]OverviewDocsEdge) {
	fromName, ok := idToServiceName[de.From]
	if !ok {
		return
	}
	toName, ok := idToServiceName[de.To]
	if !ok {
		return
	}

	fromNode, ok := serviceToNode[fromName]
	if !ok {
		return
	}
	toNode, ok := serviceToNode[toName]
	if !ok {
		return
	}

	if fromNode.ID == toNode.ID {
		return
	}

	// Adjust node IDs for internal services
	fromID := fromNode.ID
	toID := toNode.ID
	if fromNode.Internal {
		fromID = "internal." + fromNode.ID
	}
	if toNode.Internal {
		toID = "internal." + toNode.ID
	}

	key := fmt.Sprintf("%s|%s|%s|async", fromID, toID, de.Label)
	edgeSet[key] = OverviewDocsEdge{From: fromID, To: toID, Label: de.Label}
}

func findSystemServices(schema holydocs.Schema, systemName string) []holydocs.Service {
	systemServices := make([]holydocs.Service, 0)
	for _, service := range schema.Services {
		if strings.TrimSpace(service.Info.System) == systemName {
			systemServices = append(systemServices, service)
		}
	}

	return systemServices
}

func buildSystemNodes(systemServices []holydocs.Service) (map[string]SystemDocsNode,
	map[string]SystemDocsNode, map[string]string) {
	serviceToNode := make(map[string]SystemDocsNode)
	nodes := make(map[string]SystemDocsNode)
	idToServiceName := make(map[string]string, len(systemServices))

	for _, service := range systemServices {
		idToServiceName[serviceNodeID(service.Info.Name)] = service.Info.Name
	}

	for _, service := range systemServices {
		nodeID := serviceNodeID(service.Info.Name)
		content := buildOverviewNodeContent(service.Info)
		node := SystemDocsNode{
			ID:      nodeID,
			Label:   service.Info.Name,
			Content: content,
		}
		nodes[nodeID] = node
		serviceToNode[service.Info.Name] = node
	}

	return serviceToNode, nodes, idToServiceName
}

func processSystemRelationships(schema holydocs.Schema, systemServices []holydocs.Service,
	serviceToNode map[string]SystemDocsNode, nodes map[string]SystemDocsNode, edgeSet map[string]SystemDocsEdge) {
	allowedActions := map[holydocs.RelationshipAction]struct{}{
		holydocs.RelationshipActionRequests: {},
		holydocs.RelationshipActionReplies:  {},
		holydocs.RelationshipActionSends:    {},
		holydocs.RelationshipActionReceives: {},
	}

	for _, service := range systemServices {
		srcNode, ok := serviceToNode[service.Info.Name]
		if !ok {
			continue
		}

		for _, rel := range service.Relationships {
			if _, allowed := allowedActions[rel.Action]; !allowed {
				continue
			}

			tgtNode, found := findSystemTargetNode(rel, systemServices, serviceToNode, schema, nodes)
			if !found {
				continue
			}

			processSystemEdge(srcNode, tgtNode, rel, edgeSet)
		}
	}
}

func findSystemTargetNode(rel holydocs.Relationship, systemServices []holydocs.Service,
	serviceToNode map[string]SystemDocsNode, schema holydocs.Schema,
	nodes map[string]SystemDocsNode) (SystemDocsNode, bool) {
	// Check if the target service is also in this system
	for _, targetService := range systemServices {
		if targetService.Info.Name == rel.Participant {
			tgtNode, found := serviceToNode[targetService.Info.Name]

			return tgtNode, found
		}
	}

	// Check if this is an external relationship (person or external service)
	if rel.External || rel.Person {
		return getOrCreateExternalNode(rel, nodes)
	}

	// Check if this is a service from another system
	return getOrCreateOtherSystemNode(rel, schema, nodes)
}

func getOrCreateExternalNode(rel holydocs.Relationship, nodes map[string]SystemDocsNode) (SystemDocsNode, bool) {
	nodeID := externalNodeID(rel.Participant)
	node, exists := nodes[nodeID]
	if !exists {
		node = SystemDocsNode{
			ID:       nodeID,
			Label:    rel.Participant,
			Content:  formatOverviewDescription(rel.Description),
			External: true,
			Person:   rel.Person,
		}
		nodes[nodeID] = node
	}

	return node, true
}

func getOrCreateOtherSystemNode(rel holydocs.Relationship, schema holydocs.Schema,
	nodes map[string]SystemDocsNode) (SystemDocsNode, bool) {
	for _, otherService := range schema.Services {
		if otherService.Info.Name == rel.Participant {
			nodeID := serviceNodeID(rel.Participant)
			node, exists := nodes[nodeID]
			if !exists {
				content := buildOverviewNodeContent(otherService.Info)
				node = SystemDocsNode{
					ID:       nodeID,
					Label:    otherService.Info.Name,
					Content:  content,
					External: false, // This is a service from another system, not external
					Person:   false,
				}
				nodes[nodeID] = node
			}

			return node, true
		}
	}

	return SystemDocsNode{}, false
}

func processSystemEdge(srcNode, tgtNode SystemDocsNode, rel holydocs.Relationship, edgeSet map[string]SystemDocsEdge) {
	from, to := orientedEdge(srcNode.ID, tgtNode.ID, rel.Action)
	if from == to {
		return
	}

	label := string(rel.Action)
	if rel.Action == holydocs.RelationshipActionReplies {
		label = requestsLabel
	}
	key := fmt.Sprintf("%s|%s|%s|rel", from, to, label)
	edgeSet[key] = SystemDocsEdge{From: from, To: to, Label: label}
}

func processExternalServiceRelationships(schema holydocs.Schema, systemServices []holydocs.Service,
	serviceToNode map[string]SystemDocsNode, nodes map[string]SystemDocsNode,
	edgeSet map[string]SystemDocsEdge, allowedActions map[holydocs.RelationshipAction]struct{}) {
	for _, otherService := range schema.Services {
		if isServiceInSystem(otherService, systemServices) {
			continue
		}

		processOtherServiceRelationships(otherService, systemServices, serviceToNode, nodes, edgeSet, allowedActions)
	}
}

func isServiceInSystem(service holydocs.Service, systemServices []holydocs.Service) bool {
	for _, systemService := range systemServices {
		if systemService.Info.Name == service.Info.Name {
			return true
		}
	}

	return false
}

func processOtherServiceRelationships(otherService holydocs.Service, systemServices []holydocs.Service,
	serviceToNode map[string]SystemDocsNode, nodes map[string]SystemDocsNode,
	edgeSet map[string]SystemDocsEdge, allowedActions map[holydocs.RelationshipAction]struct{}) {
	for _, rel := range otherService.Relationships {
		if _, allowed := allowedActions[rel.Action]; !allowed {
			continue
		}

		tgtNode, found := findTargetInSystem(rel, systemServices, serviceToNode)
		if !found {
			continue
		}

		srcNode := getOrCreateSourceNode(otherService, nodes)
		processSystemEdge(srcNode, tgtNode, rel, edgeSet)
	}
}

func findTargetInSystem(rel holydocs.Relationship, systemServices []holydocs.Service,
	serviceToNode map[string]SystemDocsNode) (SystemDocsNode, bool) {
	for _, systemService := range systemServices {
		if systemService.Info.Name == rel.Participant {
			tgtNode, found := serviceToNode[systemService.Info.Name]

			return tgtNode, found
		}
	}

	return SystemDocsNode{}, false
}

func getOrCreateSourceNode(otherService holydocs.Service, nodes map[string]SystemDocsNode) SystemDocsNode {
	srcNodeID := serviceNodeID(otherService.Info.Name)
	srcNode, exists := nodes[srcNodeID]
	if !exists {
		content := buildOverviewNodeContent(otherService.Info)
		srcNode = SystemDocsNode{
			ID:       srcNodeID,
			Label:    otherService.Info.Name,
			Content:  content,
			External: false, // This is a service from another system, not external
			Person:   false,
		}
		nodes[srcNodeID] = srcNode
	}

	return srcNode
}

func processPersonRelationships(schema holydocs.Schema, systemServices []holydocs.Service,
	serviceToNode map[string]SystemDocsNode, nodes map[string]SystemDocsNode,
	edgeSet map[string]SystemDocsEdge, allowedActions map[holydocs.RelationshipAction]struct{}) {
	for _, service := range schema.Services {
		if !isServiceInSystem(service, systemServices) {
			continue
		}

		processServicePersonRelationships(service, serviceToNode, nodes, edgeSet, allowedActions)
	}
}

func processServicePersonRelationships(service holydocs.Service, serviceToNode map[string]SystemDocsNode,
	nodes map[string]SystemDocsNode, edgeSet map[string]SystemDocsEdge,
	allowedActions map[holydocs.RelationshipAction]struct{}) {
	for _, rel := range service.Relationships {
		if _, allowed := allowedActions[rel.Action]; !allowed {
			continue
		}

		if !rel.Person {
			continue
		}

		node := getOrCreatePersonNode(rel, nodes)
		srcNode, ok := serviceToNode[service.Info.Name]
		if !ok {
			continue
		}

		processSystemEdge(srcNode, node, rel, edgeSet)
	}
}

func getOrCreatePersonNode(rel holydocs.Relationship, nodes map[string]SystemDocsNode) SystemDocsNode {
	nodeID := externalNodeID(rel.Participant)
	node, exists := nodes[nodeID]
	if !exists {
		node = SystemDocsNode{
			ID:       nodeID,
			Label:    rel.Participant,
			Content:  formatOverviewDescription(rel.Description),
			External: true,
			Person:   true,
		}
		nodes[nodeID] = node
	}

	return node
}

func processSystemAsyncEdges(systemServices []holydocs.Service, asyncEdges []AsyncEdge,
	serviceToNode map[string]SystemDocsNode, idToServiceName map[string]string,
	edgeSet map[string]SystemDocsEdge, t *Target) map[string][]AsyncEdge {
	serviceNames := make(map[string]struct{}, len(systemServices))
	for _, svc := range systemServices {
		serviceNames[svc.Info.Name] = struct{}{}
	}

	edgesByService := buildEdgesByServiceMap(asyncEdges)

	for _, service := range systemServices {
		diagEdges, _ := t.AggregateAsyncEdgesForService(service.Info.Name, edgesByService[service.Info.Name], serviceNames)
		for _, de := range diagEdges {
			processSystemAsyncEdge(de, idToServiceName, serviceToNode, edgeSet)
		}
	}

	return edgesByService
}

func processSystemAsyncEdge(de DiagramEdge, idToServiceName map[string]string,
	serviceToNode map[string]SystemDocsNode, edgeSet map[string]SystemDocsEdge) {
	fromName, ok := idToServiceName[de.From]
	if !ok {
		return
	}
	toName, ok := idToServiceName[de.To]
	if !ok {
		return
	}

	fromNode, ok := serviceToNode[fromName]
	if !ok {
		return
	}
	toNode, ok := serviceToNode[toName]
	if !ok {
		return
	}

	if fromNode.ID == toNode.ID {
		return
	}

	key := fmt.Sprintf("%s|%s|%s|async", fromNode.ID, toNode.ID, de.Label)
	edgeSet[key] = SystemDocsEdge{From: fromNode.ID, To: toNode.ID, Label: de.Label}
}

func processExternalAsyncEdges(schema holydocs.Schema, systemServices []holydocs.Service,
	serviceToNode map[string]SystemDocsNode, nodes map[string]SystemDocsNode,
	edgeSet map[string]SystemDocsEdge, edgesByService map[string][]AsyncEdge) {
	for _, otherService := range schema.Services {
		if isServiceInSystem(otherService, systemServices) {
			continue
		}

		processOtherServiceAsyncEdges(otherService, systemServices, serviceToNode, nodes, edgeSet, edgesByService)
	}
}

func processOtherServiceAsyncEdges(otherService holydocs.Service, systemServices []holydocs.Service,
	serviceToNode map[string]SystemDocsNode, nodes map[string]SystemDocsNode,
	edgeSet map[string]SystemDocsEdge, edgesByService map[string][]AsyncEdge) {
	for _, edge := range edgesByService[otherService.Info.Name] {
		tgtNode, found := findTargetInSystemByEdge(edge, systemServices, serviceToNode)
		if !found {
			continue
		}

		srcNode := getOrCreateSourceNode(otherService, nodes)
		if srcNode.ID == tgtNode.ID {
			continue
		}

		key := fmt.Sprintf("%s|%s|%s|async", srcNode.ID, tgtNode.ID, edge.Kind)
		edgeSet[key] = SystemDocsEdge{From: srcNode.ID, To: tgtNode.ID, Label: edge.Kind}
	}
}

func findTargetInSystemByEdge(edge AsyncEdge, systemServices []holydocs.Service,
	serviceToNode map[string]SystemDocsNode) (SystemDocsNode, bool) {
	for _, systemService := range systemServices {
		if systemService.Info.Name == edge.Target {
			tgtNode, found := serviceToNode[systemService.Info.Name]

			return tgtNode, found
		}
	}

	return SystemDocsNode{}, false
}

func buildOverviewNodeContent(info holydocs.ServiceInfo) string {
	description := strings.TrimSpace(info.Description)
	if description == "" {
		return ""
	}

	return formatOverviewDescription(description)
}

func buildOverviewPayload(payload *OverviewDocsPayload, nodes map[string]OverviewDocsNode,
	edgeSet map[string]OverviewDocsEdge, globalName string) {
	nodeOrder := make([]OverviewDocsNode, 0, len(nodes))
	for _, node := range nodes {
		nodeOrder = append(nodeOrder, node)
	}

	sort.Slice(nodeOrder, func(i, j int) bool {
		if nodeOrder[i].IsSystem != nodeOrder[j].IsSystem {
			return nodeOrder[i].IsSystem && !nodeOrder[j].IsSystem
		}

		return strings.ToLower(nodeOrder[i].Label) < strings.ToLower(nodeOrder[j].Label)
	})

	edges := make([]OverviewDocsEdge, 0, len(edgeSet))
	for _, edge := range edgeSet {
		edges = append(edges, edge)
	}

	sort.Slice(edges, func(i, j int) bool {
		if edges[i].From != edges[j].From {
			return edges[i].From < edges[j].From
		}
		if edges[i].To != edges[j].To {
			return edges[i].To < edges[j].To
		}

		return edges[i].Label < edges[j].Label
	})

	// Check if we have any internal services
	hasInternalServices := false
	for _, node := range nodeOrder {
		if node.Internal {
			hasInternalServices = true

			break
		}
	}

	payload.Nodes = nodeOrder
	payload.Edges = edges
	payload.HasInternalServices = hasInternalServices

	// Set default value if globalName is empty
	if globalName == "" {
		payload.GlobalName = "Internal Services"
	} else {
		payload.GlobalName = globalName
	}
}

func buildSystemPayload(payload *SystemDocsPayload, nodes map[string]SystemDocsNode,
	edgeSet map[string]SystemDocsEdge, systemServices []holydocs.Service) {
	// Separate system nodes from external nodes
	systemNodeOrder := make([]SystemDocsNode, 0)
	externalNodeOrder := make([]SystemDocsNode, 0)

	for _, node := range nodes {
		// Check if this node belongs to the system
		isSystemNode := false
		for _, systemService := range systemServices {
			if serviceNodeID(systemService.Info.Name) == node.ID {
				isSystemNode = true

				break
			}
		}

		if isSystemNode {
			systemNodeOrder = append(systemNodeOrder, node)
		} else {
			externalNodeOrder = append(externalNodeOrder, node)
		}
	}

	// Sort system nodes
	sort.Slice(systemNodeOrder, func(i, j int) bool {
		return strings.ToLower(systemNodeOrder[i].Label) < strings.ToLower(systemNodeOrder[j].Label)
	})

	// Sort external nodes
	sort.Slice(externalNodeOrder, func(i, j int) bool {
		return strings.ToLower(externalNodeOrder[i].Label) < strings.ToLower(externalNodeOrder[j].Label)
	})

	edges := make([]SystemDocsEdge, 0, len(edgeSet))
	for _, edge := range edgeSet {
		edges = append(edges, edge)
	}

	sort.Slice(edges, func(i, j int) bool {
		if edges[i].From != edges[j].From {
			return edges[i].From < edges[j].From
		}
		if edges[i].To != edges[j].To {
			return edges[i].To < edges[j].To
		}

		return edges[i].Label < edges[j].Label
	})

	payload.SystemNodes = systemNodeOrder
	payload.ExternalNodes = externalNodeOrder
	payload.Edges = edges
}

func orientedEdge(sourceID, targetID string, action holydocs.RelationshipAction) (string, string) {
	switch action {
	case holydocs.RelationshipActionReceives, holydocs.RelationshipActionReplies:
		return targetID, sourceID
	default:
		return sourceID, targetID
	}
}
