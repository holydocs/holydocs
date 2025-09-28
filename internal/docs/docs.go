package docs

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
	"unicode"

	"github.com/holydocs/holydocs/pkg/holydocs"
	d2target "github.com/holydocs/holydocs/pkg/schema/target/d2"
	mf "github.com/holydocs/messageflow/pkg/messageflow"
)

// Errors.
var (
	ErrHolydocsTargetRequired  = errors.New("holydocs target is required")
	ErrDirectoryCreationFailed = errors.New("failed to create directory")
	ErrTemplateExecutionFailed = errors.New("failed to execute template")
	ErrFileWriteFailed         = errors.New("failed to write file")
)

//go:embed templates/readme.tmpl
var readmeTemplateFS embed.FS

// File permissions.
const (
	dirPerm  = 0o755
	filePerm = 0o644
)

// Directory names.
const (
	diagramsDirName           = "diagrams"
	servicesDiagramDirName    = "services"
	messageflowDiagramDirName = "messageflow"
)

type templateData struct {
	Title           string
	OverviewDiagram string
	OverviewD2      string
	Systems         []systemView
	SystemDiagrams  map[string]systemDiagramView
	MessageFlow     messageFlowView
}

type systemView struct {
	Name     string
	Anchor   string
	Services []serviceView
}

type systemDiagramView struct {
	SystemDiagram string
	SystemD2      string
	SystemD2Name  string
}

type serviceView struct {
	Name                  string
	Anchor                string
	System                string
	Description           string
	Owner                 string
	Repository            string
	Tags                  []string
	RelationshipsDiagram  string
	RelationshipsD2       string
	RelationshipSummaries []relationshipSummary
	InterServiceLinks     []serviceConnection
	AsyncSummaries        []asyncSummary
	ServiceFlowDiagram    string
}

type relationshipSummary struct {
	Participant string
	Action      holydocs.RelationshipAction
	Technology  string
	Description string
	Proto       string
	External    bool
	Person      bool
}

type serviceConnection struct {
	Direction string
	Target    string
	Channel   string
	Kind      string
}

type asyncSummary struct {
	Direction string
	Target    string
	Label     string
}

type messageFlowView struct {
	HasData        bool
	ContextDiagram string
	Channels       []channelView
}

type channelView struct {
	Name        string
	Anchor      string
	DiagramPath string
	Messages    []channelMessage
}

type channelMessage struct {
	Name      string
	Direction string
	Payload   string
}

type asyncEdge struct {
	Source  string
	Target  string
	Channel string
	Kind    string
}

// Convert asyncEdge to d2target.AsyncEdge.
func (ae asyncEdge) toD2AsyncEdge() d2target.AsyncEdge {
	return d2target.AsyncEdge{
		Source:  ae.Source,
		Target:  ae.Target,
		Channel: ae.Channel,
		Kind:    ae.Kind,
	}
}

// Convert []asyncEdge to []d2target.AsyncEdge.
func convertAsyncEdges(edges []asyncEdge) []d2target.AsyncEdge {
	result := make([]d2target.AsyncEdge, len(edges))
	for i, edge := range edges {
		result[i] = edge.toD2AsyncEdge()
	}

	return result
}

// Generate produces the documentation bundle (markdown + diagrams) for the provided schemas.
func Generate(
	ctx context.Context,
	schema holydocs.Schema,
	holydocsTarget holydocs.Target,
	messageflowSchema mf.Schema,
	messageflowTarget mf.Target,
	title, globalName, outputDir string,
) error {
	if holydocsTarget == nil {
		return ErrHolydocsTargetRequired
	}

	diagramsDir, serviceDiagramDir, messageflowDiagramDir, err := setupOutputDirectories(outputDir)
	if err != nil {
		return err
	}

	schema.Sort()
	messageflowSchema.Sort()
	asyncEdges := buildAsyncEdges(messageflowSchema)

	overviewDiagramPath, serviceViews, systemDiagrams, mfv, err := generateAllDiagrams(
		ctx, schema, asyncEdges, holydocsTarget, messageflowSchema, messageflowTarget,
		globalName, diagramsDir, serviceDiagramDir, messageflowDiagramDir)
	if err != nil {
		return err
	}

	data := buildTemplateData(title, overviewDiagramPath, serviceViews, systemDiagrams, mfv)

	return writeReadme(outputDir, data)
}

func setupOutputDirectories(outputDir string) (string, string, string, error) {
	if err := os.MkdirAll(outputDir, dirPerm); err != nil {
		return "", "", "", fmt.Errorf("%w: %w", ErrDirectoryCreationFailed, err)
	}

	diagramsDir := filepath.Join(outputDir, diagramsDirName)
	if err := os.RemoveAll(diagramsDir); err != nil {
		return "", "", "", fmt.Errorf("failed to clean diagrams directory: %w", err)
	}

	if err := os.MkdirAll(diagramsDir, dirPerm); err != nil {
		return "", "", "", fmt.Errorf("%w diagrams directory: %w", ErrDirectoryCreationFailed, err)
	}

	serviceDiagramDir := filepath.Join(diagramsDir, servicesDiagramDirName)
	if err := os.MkdirAll(serviceDiagramDir, dirPerm); err != nil {
		return "", "", "", fmt.Errorf("%w service diagrams directory: %w", ErrDirectoryCreationFailed, err)
	}

	messageflowDiagramDir := filepath.Join(diagramsDir, messageflowDiagramDirName)
	if err := os.MkdirAll(messageflowDiagramDir, dirPerm); err != nil {
		return "", "", "", fmt.Errorf("%w message flow diagrams directory: %w", ErrDirectoryCreationFailed, err)
	}

	return diagramsDir, serviceDiagramDir, messageflowDiagramDir, nil
}

func generateAllDiagrams(
	ctx context.Context,
	schema holydocs.Schema,
	asyncEdges []asyncEdge,
	holydocsTarget holydocs.Target,
	messageflowSchema mf.Schema,
	messageflowTarget mf.Target,
	globalName, diagramsDir, serviceDiagramDir, messageflowDiagramDir string,
) (string, []serviceView, map[string]systemDiagramView, messageFlowView, error) {
	overviewDiagramPath := filepath.Join(diagramsDir, "overview.svg")
	if err := generateOverviewDiagram(ctx, schema, asyncEdges, holydocsTarget, globalName,
		overviewDiagramPath); err != nil {
		return "", nil, nil, messageFlowView{}, fmt.Errorf("failed to generate overview diagram: %w", err)
	}

	serviceViews, err := buildServiceViews(ctx, schema, asyncEdges, holydocsTarget,
		messageflowSchema, messageflowTarget, serviceDiagramDir)
	if err != nil {
		return "", nil, nil, messageFlowView{}, fmt.Errorf("failed to build service views: %w", err)
	}

	systemDiagrams, err := generateSystemDiagrams(ctx, schema, asyncEdges, holydocsTarget, diagramsDir)
	if err != nil {
		return "", nil, nil, messageFlowView{}, fmt.Errorf("failed to generate system diagrams: %w", err)
	}

	mfv, err := generateMessageFlowSection(ctx, messageflowSchema, messageflowTarget, messageflowDiagramDir)
	if err != nil {
		return "", nil, nil, messageFlowView{}, fmt.Errorf("failed to generate message flow diagrams: %w", err)
	}

	return overviewDiagramPath, serviceViews, systemDiagrams, mfv, nil
}

func buildTemplateData(title, overviewDiagramPath string, serviceViews []serviceView,
	systemDiagrams map[string]systemDiagramView, mfv messageFlowView) templateData {
	return templateData{
		Title:           title,
		OverviewDiagram: filepath.ToSlash(filepath.Join(diagramsDirName, filepath.Base(overviewDiagramPath))),
		OverviewD2: filepath.ToSlash(filepath.Join(diagramsDirName,
			strings.TrimSuffix(filepath.Base(overviewDiagramPath), ".svg")+".d2")),
		Systems:        groupServicesBySystem(serviceViews),
		SystemDiagrams: systemDiagrams,
		MessageFlow:    mfv,
	}
}

func generateSystemDiagrams(
	ctx context.Context,
	schema holydocs.Schema,
	asyncEdges []asyncEdge,
	target holydocs.Target,
	diagramsDir string,
) (map[string]systemDiagramView, error) {
	// Convert to D2 target
	d2Target, ok := target.(*d2target.Target)
	if !ok {
		return nil, errors.New("target is not a D2 target")
	}

	// Find all unique systems
	systems := make(map[string]struct{})
	for _, service := range schema.Services {
		if systemName := strings.TrimSpace(service.Info.System); systemName != "" {
			systems[systemName] = struct{}{}
		}
	}

	systemDiagrams := make(map[string]systemDiagramView)

	// Generate diagrams for each system
	for systemName := range systems {
		// Generate D2 script
		script, err := d2Target.GenerateSystemDiagramScript(schema, systemName, convertAsyncEdges(asyncEdges))
		if err != nil {
			return nil, fmt.Errorf("generate system D2 script for %s: %w", systemName, err)
		}

		if len(script) == 0 {
			continue // Skip systems with no content
		}

		// Save raw D2 script
		d2Filename := fmt.Sprintf("system-%s.d2", sanitizeFilename(systemName))
		d2Path := filepath.Join(diagramsDir, d2Filename)
		if err := os.WriteFile(d2Path, script, filePerm); err != nil {
			return nil, fmt.Errorf("write system D2 script for %s: %w", systemName, err)
		}

		// Generate SVG diagram
		diagram, err := d2Target.GenerateSystemDiagram(ctx, schema, systemName, convertAsyncEdges(asyncEdges))
		if err != nil {
			return nil, fmt.Errorf("render system diagram for %s: %w", systemName, err)
		}

		// Save SVG diagram
		svgFilename := fmt.Sprintf("system-%s.svg", sanitizeFilename(systemName))
		svgPath := filepath.Join(diagramsDir, svgFilename)
		if err := os.WriteFile(svgPath, diagram, filePerm); err != nil {
			return nil, fmt.Errorf("write system diagram for %s: %w", systemName, err)
		}

		// Add to system diagrams map using display name
		displayName := systemName
		if displayName == "" {
			displayName = "Standalone Services"
		}
		systemDiagrams[displayName] = systemDiagramView{
			SystemDiagram: filepath.ToSlash(filepath.Join(diagramsDirName, svgFilename)),
			SystemD2:      filepath.ToSlash(filepath.Join(diagramsDirName, d2Filename)),
			SystemD2Name:  d2Filename,
		}
	}

	return systemDiagrams, nil
}

func buildServiceViews(
	ctx context.Context,
	schema holydocs.Schema,
	asyncEdges []asyncEdge,
	holydocsTarget holydocs.Target,
	messageflowSchema mf.Schema,
	messageflowTarget mf.Target,
	outputDir string,
) ([]serviceView, error) {
	serviceNameSet := buildServiceNameSet(schema.Services)
	edgesByService := buildEdgesByServiceMap(asyncEdges)

	views := make([]serviceView, 0, len(schema.Services))
	for _, service := range schema.Services {
		view, err := buildServiceView(ctx, service, schema.Services, edgesByService,
			holydocsTarget, messageflowSchema, messageflowTarget, serviceNameSet, outputDir)
		if err != nil {
			return nil, err
		}
		views = append(views, view)
	}

	sort.SliceStable(views, func(i, j int) bool {
		if strings.EqualFold(views[i].Name, views[j].Name) {
			return views[i].Name < views[j].Name
		}

		return strings.ToLower(views[i].Name) < strings.ToLower(views[j].Name)
	})

	return views, nil
}

func buildServiceNameSet(services []holydocs.Service) map[string]struct{} {
	serviceNameSet := make(map[string]struct{}, len(services))
	for _, service := range services {
		serviceNameSet[service.Info.Name] = struct{}{}
	}

	return serviceNameSet
}

func buildEdgesByServiceMap(asyncEdges []asyncEdge) map[string][]asyncEdge {
	edgesByService := make(map[string][]asyncEdge)
	for _, edge := range asyncEdges {
		edgesByService[edge.Source] = append(edgesByService[edge.Source], edge)
		edgesByService[edge.Target] = append(edgesByService[edge.Target], edge)
	}

	return edgesByService
}

func buildServiceView(
	ctx context.Context,
	service holydocs.Service,
	allServices []holydocs.Service,
	edgesByService map[string][]asyncEdge,
	holydocsTarget holydocs.Target,
	messageflowSchema mf.Schema,
	messageflowTarget mf.Target,
	serviceNameSet map[string]struct{},
	outputDir string,
) (serviceView, error) {
	filenameBase := sanitizeFilename(service.Info.Name)

	relationshipDiagram := filepath.Join(outputDir, filenameBase+"-relationships.svg")
	if err := generateServiceRelationshipsDiagram(ctx, service, allServices,
		edgesByService[service.Info.Name], holydocsTarget, relationshipDiagram); err != nil {
		return serviceView{}, err
	}

	asyncSummaries := buildAsyncSummaries(service.Info.Name, edgesByService, holydocsTarget, serviceNameSet)
	serviceFlowDiagram := buildServiceFlowDiagram(ctx, service, messageflowSchema,
		messageflowTarget, outputDir, filenameBase)

	tags := append([]string(nil), service.Info.Tags...)
	sort.Strings(tags)

	return serviceView{
		Name:        service.Info.Name,
		Anchor:      sanitizeAnchor(service.Info.Name),
		System:      service.Info.System,
		Description: d2target.FormatDescription(strings.TrimSpace(service.Info.Description)),
		Owner:       service.Info.Owner,
		Repository:  service.Info.Repository,
		Tags:        tags,
		RelationshipsDiagram: filepath.ToSlash(filepath.Join(diagramsDirName,
			servicesDiagramDirName, filepath.Base(relationshipDiagram))),
		RelationshipsD2: filepath.ToSlash(filepath.Join(diagramsDirName,
			servicesDiagramDirName, strings.TrimSuffix(filepath.Base(relationshipDiagram), ".svg")+".d2")),
		RelationshipSummaries: buildRelationshipSummaries(service.Relationships),
		InterServiceLinks:     buildServiceConnections(service.Info.Name, edgesByService[service.Info.Name]),
		AsyncSummaries:        asyncSummaries,
		ServiceFlowDiagram:    serviceFlowDiagram,
	}, nil
}

func buildAsyncSummaries(serviceName string, edgesByService map[string][]asyncEdge,
	holydocsTarget holydocs.Target, serviceNameSet map[string]struct{}) []asyncSummary {
	if len(edgesByService[serviceName]) == 0 {
		return nil
	}

	d2Target, ok := holydocsTarget.(*d2target.Target)
	if !ok {
		return nil
	}

	_, summaries := d2Target.AggregateAsyncEdgesForService(serviceName,
		convertAsyncEdges(edgesByService[serviceName]), serviceNameSet)
	if len(summaries) == 0 {
		return nil
	}

	asyncSummaries := make([]asyncSummary, len(summaries))
	for i, summary := range summaries {
		asyncSummaries[i] = asyncSummary{
			Direction: summary.Direction,
			Target:    summary.Target,
			Label:     summary.Label,
		}
	}

	sort.SliceStable(asyncSummaries, func(i, j int) bool {
		if !strings.EqualFold(asyncSummaries[i].Target, asyncSummaries[j].Target) {
			return strings.ToLower(asyncSummaries[i].Target) < strings.ToLower(asyncSummaries[j].Target)
		}

		return asyncSummaries[i].Direction < asyncSummaries[j].Direction
	})

	return asyncSummaries
}

func buildServiceFlowDiagram(ctx context.Context, service holydocs.Service,
	messageflowSchema mf.Schema, messageflowTarget mf.Target, outputDir, filenameBase string) string {
	if messageflowTarget == nil || len(messageflowSchema.Services) == 0 {
		return ""
	}

	servicesDiagramPath := filepath.Join(outputDir, filenameBase+"-service-services.svg")
	err := generateMessageFlowDiagram(ctx, messageflowSchema, messageflowTarget, mf.FormatOptions{
		Mode:    mf.FormatModeServiceServices,
		Service: service.Info.Name,
	}, servicesDiagramPath)
	if err == nil {
		return filepath.ToSlash(filepath.Join(diagramsDirName,
			servicesDiagramDirName, filepath.Base(servicesDiagramPath)))
	}
	if !errors.Is(err, errNoDiagramData) {
		// Log error but don't fail the entire process
		return ""
	}

	return ""
}

func buildRelationshipSummaries(rels []holydocs.Relationship) []relationshipSummary {
	if len(rels) == 0 {
		return nil
	}

	summaries := make([]relationshipSummary, 0, len(rels))
	for _, rel := range rels {
		summaries = append(summaries, relationshipSummary{
			Participant: rel.Participant,
			Action:      rel.Action,
			Technology:  rel.Technology,
			Description: rel.Description,
			Proto:       rel.Proto,
			External:    rel.External,
			Person:      rel.Person,
		})
	}

	sort.SliceStable(summaries, func(i, j int) bool {
		if summaries[i].Action != summaries[j].Action {
			return summaries[i].Action < summaries[j].Action
		}

		return summaries[i].Participant < summaries[j].Participant
	})

	return summaries
}

func buildServiceConnections(serviceName string, edges []asyncEdge) []serviceConnection {
	if len(edges) == 0 {
		return nil
	}

	connections := make([]serviceConnection, 0, len(edges))
	seen := make(map[string]struct{})

	for _, edge := range edges {
		direction := ""
		target := ""

		switch {
		case edge.Source == serviceName && edge.Target == serviceName:
			continue
		case edge.Source == serviceName:
			direction = "sends to"
			target = edge.Target
		case edge.Target == serviceName:
			direction = "receives from"
			target = edge.Source
		}

		if direction == "" {
			continue
		}

		if edge.Kind == "reply" && edge.Source == serviceName {
			direction = "replies to"
		}

		key := fmt.Sprintf("%s|%s|%s|%s", direction, target, edge.Channel, edge.Kind)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}

		connections = append(connections, serviceConnection{
			Direction: direction,
			Target:    target,
			Channel:   edge.Channel,
			Kind:      edge.Kind,
		})
	}

	sort.SliceStable(connections, func(i, j int) bool {
		if connections[i].Target != connections[j].Target {
			return connections[i].Target < connections[j].Target
		}
		if connections[i].Direction != connections[j].Direction {
			return connections[i].Direction < connections[j].Direction
		}
		if connections[i].Kind != connections[j].Kind {
			return connections[i].Kind < connections[j].Kind
		}

		return connections[i].Channel < connections[j].Channel
	})

	return connections
}

func generateOverviewDiagram(
	ctx context.Context,
	schema holydocs.Schema,
	asyncEdges []asyncEdge,
	target holydocs.Target,
	globalName, outputPath string,
) error {
	// Convert to D2 target
	d2Target, ok := target.(*d2target.Target)
	if !ok {
		return errors.New("target is not a D2 target")
	}

	// Generate D2 script
	script, err := d2Target.GenerateOverviewDiagramScript(schema, convertAsyncEdges(asyncEdges), globalName)
	if err != nil {
		return fmt.Errorf("generate overview D2 script: %w", err)
	}

	if len(script) == 0 {
		return nil
	}

	// Save raw D2 script
	d2Path := strings.TrimSuffix(outputPath, ".svg") + ".d2"
	if err := os.WriteFile(d2Path, script, filePerm); err != nil {
		return fmt.Errorf("write overview D2 script: %w", err)
	}

	// Generate SVG diagram
	diagram, err := d2Target.GenerateOverviewDiagram(ctx, schema, convertAsyncEdges(asyncEdges), globalName)
	if err != nil {
		return fmt.Errorf("render overview diagram: %w", err)
	}

	if err := os.WriteFile(outputPath, diagram, filePerm); err != nil {
		return fmt.Errorf("write overview diagram: %w", err)
	}

	return nil
}

func generateServiceRelationshipsDiagram(
	ctx context.Context,
	service holydocs.Service,
	allServices []holydocs.Service,
	serviceEdges []asyncEdge,
	target holydocs.Target,
	outputPath string,
) error {
	// Convert to D2 target
	d2Target, ok := target.(*d2target.Target)
	if !ok {
		return errors.New("target is not a D2 target")
	}

	// Generate D2 script
	script, err := d2Target.GenerateServiceRelationshipsDiagramScript(service, allServices,
		convertAsyncEdges(serviceEdges))
	if err != nil {
		return fmt.Errorf("generate service relationships D2 script: %w", err)
	}

	if len(script) == 0 {
		return nil
	}

	// Save raw D2 script
	d2Path := strings.TrimSuffix(outputPath, ".svg") + ".d2"
	if err := os.WriteFile(d2Path, script, filePerm); err != nil {
		return fmt.Errorf("write service relationships D2 script: %w", err)
	}

	// Generate SVG diagram
	diagram, err := d2Target.GenerateServiceRelationshipsDiagram(ctx, service, allServices,
		convertAsyncEdges(serviceEdges))
	if err != nil {
		return fmt.Errorf("render service relationships diagram: %w", err)
	}

	if err := os.WriteFile(outputPath, diagram, filePerm); err != nil {
		return fmt.Errorf("write service relationships diagram: %w", err)
	}

	return nil
}

var errNoDiagramData = errors.New("no diagram data")

func generateMessageFlowDiagram(
	ctx context.Context,
	schema mf.Schema,
	target mf.Target,
	opts mf.FormatOptions,
	outputPath string,
) error {
	if target == nil || len(schema.Services) == 0 {
		return errNoDiagramData
	}

	formatted, err := target.FormatSchema(ctx, schema, opts)
	if err != nil {
		return fmt.Errorf("format schema: %w", err)
	}

	if len(formatted.Data) == 0 {
		return errNoDiagramData
	}

	diagram, err := target.RenderSchema(ctx, formatted)
	if err != nil {
		return fmt.Errorf("render schema: %w", err)
	}

	if len(diagram) == 0 {
		return errNoDiagramData
	}

	if err := os.WriteFile(outputPath, diagram, filePerm); err != nil {
		return fmt.Errorf("write diagram: %w", err)
	}

	return nil
}

func generateMessageFlowSection(
	ctx context.Context,
	schema mf.Schema,
	target mf.Target,
	outputDir string,
) (messageFlowView, error) {
	result := messageFlowView{}

	if target == nil || len(schema.Services) == 0 {
		return result, nil
	}

	contextDiagram := filepath.Join(outputDir, "context.svg")
	if err := generateMessageFlowDiagram(ctx, schema, target,
		mf.FormatOptions{Mode: mf.FormatModeContextServices}, contextDiagram); err != nil {
		if errors.Is(err, errNoDiagramData) {
			return result, nil
		}

		return result, err
	}

	channelViews, err := generateChannelViews(ctx, schema, target, outputDir)
	if err != nil {
		return result, err
	}

	if len(channelViews) == 0 {
		return result, nil
	}

	sort.SliceStable(channelViews, func(i, j int) bool {
		return channelViews[i].Name < channelViews[j].Name
	})

	result.HasData = true
	result.ContextDiagram = filepath.ToSlash(filepath.Join(diagramsDirName,
		messageflowDiagramDirName, filepath.Base(contextDiagram)))
	result.Channels = channelViews

	return result, nil
}

func generateChannelViews(
	ctx context.Context,
	schema mf.Schema,
	target mf.Target,
	outputDir string,
) ([]channelView, error) {
	channels := extractUniqueChannels(schema)
	channelInfo := extractChannelInfo(schema)
	channelViews := make([]channelView, 0, len(channels))

	for _, channel := range channels {
		filename := fmt.Sprintf("channel-%s.svg", sanitizeFilename(channel))
		path := filepath.Join(outputDir, filename)
		err := generateMessageFlowDiagram(ctx, schema, target, mf.FormatOptions{
			Mode:         mf.FormatModeChannelServices,
			Channel:      channel,
			OmitPayloads: true,
		}, path)
		if err != nil {
			if errors.Is(err, errNoDiagramData) {
				continue
			}

			return nil, fmt.Errorf("channel diagram for %s: %w", channel, err)
		}

		channelViews = append(channelViews, channelView{
			Name:        channel,
			Anchor:      sanitizeAnchor(channel),
			DiagramPath: filepath.ToSlash(filepath.Join(diagramsDirName, messageflowDiagramDirName, filename)),
			Messages:    channelInfo[channel],
		})
	}

	return channelViews, nil
}

func buildAsyncEdges(schema mf.Schema) []asyncEdge {
	if len(schema.Services) == 0 {
		return nil
	}

	channels := buildChannelParticipants(schema.Services)
	edgeSet := buildEdgeSetFromChannels(channels)
	edges := convertEdgeSetToSlice(edgeSet)
	sortAsyncEdges(edges)

	return edges
}

type participants struct {
	senders   map[string]struct{}
	receivers map[string]struct{}
	repliers  map[string]struct{}
}

func buildChannelParticipants(services []mf.Service) map[string]*participants {
	channels := make(map[string]*participants)

	for _, service := range services {
		for _, op := range service.Operation {
			channelName := op.Channel.Name
			if channelName == "" {
				continue
			}

			p := getOrCreateParticipants(channels, channelName)
			updateParticipantsForOperation(p, service.Name, op)
		}
	}

	return channels
}

func getOrCreateParticipants(channels map[string]*participants, channelName string) *participants {
	p, exists := channels[channelName]
	if !exists {
		p = &participants{
			senders:   make(map[string]struct{}),
			receivers: make(map[string]struct{}),
			repliers:  make(map[string]struct{}),
		}
		channels[channelName] = p
	}

	return p
}

func updateParticipantsForOperation(p *participants, serviceName string, op mf.Operation) {
	switch op.Action {
	case mf.ActionSend:
		p.senders[serviceName] = struct{}{}
	case mf.ActionReceive:
		p.receivers[serviceName] = struct{}{}
		if op.Reply != nil {
			p.repliers[serviceName] = struct{}{}
		}
	}
}

func buildEdgeSetFromChannels(channels map[string]*participants) map[string]asyncEdge {
	edgeSet := make(map[string]asyncEdge)

	for channel, p := range channels {
		buildSendEdges(edgeSet, channel, p)
		buildReplyEdges(edgeSet, channel, p)
	}

	return edgeSet
}

func buildSendEdges(edgeSet map[string]asyncEdge, channel string, p *participants) {
	for sender := range p.senders {
		for receiver := range p.receivers {
			if sender == receiver {
				continue
			}

			key := fmt.Sprintf("%s|%s|%s|send", sender, receiver, channel)
			edgeSet[key] = asyncEdge{
				Source:  sender,
				Target:  receiver,
				Channel: channel,
				Kind:    "send",
			}
		}
	}
}

func buildReplyEdges(edgeSet map[string]asyncEdge, channel string, p *participants) {
	for replier := range p.repliers {
		for requester := range p.senders {
			if replier == requester {
				continue
			}
			key := fmt.Sprintf("%s|%s|%s|reply", replier, requester, channel)
			edgeSet[key] = asyncEdge{
				Source:  replier,
				Target:  requester,
				Channel: channel,
				Kind:    "reply",
			}
		}
	}
}

func convertEdgeSetToSlice(edgeSet map[string]asyncEdge) []asyncEdge {
	edges := make([]asyncEdge, 0, len(edgeSet))
	for _, edge := range edgeSet {
		edges = append(edges, edge)
	}

	return edges
}

func sortAsyncEdges(edges []asyncEdge) {
	sort.SliceStable(edges, func(i, j int) bool {
		if edges[i].Source != edges[j].Source {
			return edges[i].Source < edges[j].Source
		}
		if edges[i].Target != edges[j].Target {
			return edges[i].Target < edges[j].Target
		}
		if edges[i].Channel != edges[j].Channel {
			return edges[i].Channel < edges[j].Channel
		}

		return edges[i].Kind < edges[j].Kind
	})
}

func groupServicesBySystem(services []serviceView) []systemView {
	systems := make(map[string][]serviceView)
	order := make([]string, 0)

	for _, svc := range services {
		systems[svc.System] = append(systems[svc.System], svc)
	}

	for system := range systems {
		order = append(order, system)
	}
	sort.SliceStable(order, func(i, j int) bool {
		iStandalone := order[i] == ""
		jStandalone := order[j] == ""
		if iStandalone && !jStandalone {
			return false
		}
		if !iStandalone && jStandalone {
			return true
		}

		return order[i] < order[j]
	})

	result := make([]systemView, 0, len(order))
	for _, system := range order {
		servicesInSystem := systems[system]
		sort.SliceStable(servicesInSystem, func(i, j int) bool {
			return servicesInSystem[i].Name < servicesInSystem[j].Name
		})

		displayName := system
		if displayName == "" {
			displayName = "Standalone Services"
		}

		result = append(result, systemView{
			Name:     displayName,
			Anchor:   sanitizeAnchor(displayName),
			Services: servicesInSystem,
		})
	}

	return result
}

func sanitizeAnchor(name string) string {
	anchor := strings.ToLower(strings.TrimSpace(name))
	anchor = strings.ReplaceAll(anchor, " ", "-")
	anchor = strings.ReplaceAll(anchor, "_", "-")

	var builder strings.Builder
	for _, r := range anchor {
		if unicode.IsLetter(r) || unicode.IsNumber(r) || r == '-' {
			builder.WriteRune(r)
		}
	}

	result := builder.String()
	result = strings.ReplaceAll(result, "--", "-")
	result = strings.Trim(result, "-")

	return result
}

func sanitizeFilename(name string) string {
	anchor := sanitizeAnchor(name)
	if anchor == "" {
		return "item"
	}

	return anchor
}

func writeReadme(outputDir string, data templateData) error {
	tmpl, err := template.New("readme.tmpl").Funcs(template.FuncMap{
		"Anchor": sanitizeAnchor,
		"Join":   strings.Join,
		"lower":  strings.ToLower,
	}).ParseFS(readmeTemplateFS, "templates/readme.tmpl")
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}

	readmePath := filepath.Join(outputDir, "README.md")
	if err := os.WriteFile(readmePath, []byte(buf.String()), filePerm); err != nil {
		return fmt.Errorf("write README: %w", err)
	}

	return nil
}

func extractUniqueChannels(schema mf.Schema) []string {
	channelSet := make(map[string]struct{})
	for _, service := range schema.Services {
		for _, op := range service.Operation {
			channelSet[op.Channel.Name] = struct{}{}
			if op.Reply != nil {
				channelSet[op.Reply.Name] = struct{}{}
			}
		}
	}

	channels := make([]string, 0, len(channelSet))
	for channel := range channelSet {
		channels = append(channels, channel)
	}

	sort.Strings(channels)

	return channels
}

func extractChannelInfo(schema mf.Schema) map[string][]channelMessage {
	operationsByChannel := buildOperationsByChannel(schema.Services)
	channelInfo := make(map[string][]channelMessage)

	for channelName, operations := range operationsByChannel {
		messages := extractMessagesFromOperations(operations)
		channelInfo[channelName] = messages
	}

	return channelInfo
}

type opEntry struct {
	operation mf.Operation
}

func buildOperationsByChannel(services []mf.Service) map[string][]opEntry {
	operationsByChannel := make(map[string][]opEntry)

	for _, service := range services {
		for _, operation := range service.Operation {
			operationsByChannel[operation.Channel.Name] = append(
				operationsByChannel[operation.Channel.Name], opEntry{operation: operation})
		}
	}

	return operationsByChannel
}

func extractMessagesFromOperations(operations []opEntry) []channelMessage {
	messages := make([]channelMessage, 0)

	appendMessage := func(direction string, msg mf.Message) {
		if msg.Name == "" && strings.TrimSpace(msg.Payload) == "" {
			return
		}
		messages = append(messages, channelMessage{
			Name:      msg.Name,
			Direction: direction,
			Payload:   msg.Payload,
		})
	}

	if hasReplyOperation(operations) {
		extractReplyMessages(operations, appendMessage)
	} else {
		extractNonReplyMessages(operations, appendMessage)
	}

	return messages
}

func hasReplyOperation(operations []opEntry) bool {
	for _, op := range operations {
		if op.operation.Reply != nil {
			return true
		}
	}

	return false
}

func extractReplyMessages(operations []opEntry, appendMessage func(string, mf.Message)) {
	for _, op := range operations {
		if op.operation.Reply == nil {
			continue
		}
		appendMessage("request", op.operation.Channel.Message)
		appendMessage("reply", op.operation.Reply.Message)

		break
	}
}

func extractNonReplyMessages(operations []opEntry, appendMessage func(string, mf.Message)) {
	if extractReceiveMessages(operations, appendMessage) {
		return
	}
	extractSendMessages(operations, appendMessage)
}

func extractReceiveMessages(operations []opEntry, appendMessage func(string, mf.Message)) bool {
	for _, op := range operations {
		if op.operation.Action != mf.ActionReceive {
			continue
		}
		appendMessage("receive", op.operation.Channel.Message)

		return true
	}

	return false
}

func extractSendMessages(operations []opEntry, appendMessage func(string, mf.Message)) {
	for _, op := range operations {
		if op.operation.Action != mf.ActionSend {
			continue
		}
		appendMessage("send", op.operation.Channel.Message)

		break
	}
}
