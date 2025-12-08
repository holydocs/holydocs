package docs

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
	"unicode"

	d2target "github.com/holydocs/holydocs/internal/adapters/secondary/target/d2"
	"github.com/holydocs/holydocs/internal/config"
	"github.com/holydocs/holydocs/internal/core/domain"
	mf "github.com/holydocs/messageflow/pkg/messageflow"
	do "github.com/samber/do/v2"
)

// Errors.
var (
	ErrHolydocsTargetRequired  = errors.New("holydocs target is required")
	ErrDirectoryCreationFailed = errors.New("failed to create directory")
)

//go:embed templates/readme.tmpl
var readmeTemplateFS embed.FS

// DocumentationConfig is an alias for config.Documentation to avoid circular imports.
type DocumentationConfig = config.Documentation

// Metadata represents the metadata for the documentation.
type Metadata struct {
	Schema     domain.Schema      `json:"schema"`
	Changelogs []domain.Changelog `json:"changelogs"`
}

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
	Title            string
	OverviewDiagram  string
	OverviewD2       string
	OverviewMarkdown string
	Systems          []systemView
	SystemDiagrams   map[string]systemDiagramView
	SystemMarkdowns  map[string]string
	ServiceSummaries map[string]string
	SystemSummaries  map[string]string
	MessageFlow      messageFlowView
	Changelogs       []domain.Changelog
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
	Action      domain.RelationshipAction
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

func (ae asyncEdge) toD2AsyncEdge() domain.AsyncEdge {
	return domain.AsyncEdge{
		Source:  ae.Source,
		Target:  ae.Target,
		Channel: ae.Channel,
		Kind:    ae.Kind,
	}
}

func convertAsyncEdges(edges []asyncEdge) []domain.AsyncEdge {
	result := make([]domain.AsyncEdge, len(edges))
	for i, edge := range edges {
		result[i] = edge.toD2AsyncEdge()
	}

	return result
}

// Generator implements the DocumentationGenerator interface.
type Generator struct{}

func NewGenerator(_ do.Injector) (*Generator, error) {
	return &Generator{}, nil
}

// Generate produces the documentation bundle (markdown + diagrams) for the provided schemas.
func (g *Generator) Generate(
	ctx context.Context,
	schema domain.Schema,
	holydocsTarget domain.Target,
	messageflowSchema mf.Schema,
	messageflowTarget mf.Target,
	cfg *config.Config,
) (*domain.Changelog, error) {
	if holydocsTarget == nil {
		return nil, ErrHolydocsTargetRequired
	}

	// Sort schemas before processing to ensure consistent ordering
	schema.Sort()
	messageflowSchema.Sort()

	metadata, newChangelog, err := g.processMetadata(schema, cfg.Output.Dir)
	if err != nil {
		return nil, fmt.Errorf("error processing metadata: %w", err)
	}

	outputDirs, err := setupOutputDirectories(cfg.Output.Dir)
	if err != nil {
		return nil, err
	}

	asyncEdges := buildAsyncEdges(messageflowSchema)

	diagramResults, err := generateAllDiagrams(
		ctx, schema, asyncEdges, holydocsTarget, messageflowSchema, messageflowTarget, cfg, outputDirs)
	if err != nil {
		return nil, err
	}

	data := buildTemplateData(cfg, diagramResults, metadata.Changelogs)

	return newChangelog, writeReadme(cfg.Output.Dir, data)
}

func (g *Generator) processMetadata(schema domain.Schema, outputDir string) (*Metadata, *domain.Changelog, error) {
	existingMetadata, err := readMetadata(outputDir)
	if err != nil {
		return nil, nil, fmt.Errorf("error reading existing holydocs data: %w", err)
	}

	var (
		newChangelog       *domain.Changelog
		existingChangelogs []domain.Changelog
	)

	if existingMetadata != nil {
		changelog := existingMetadata.Schema.Compare(schema)
		if len(changelog.Changes) > 0 {
			newChangelog = &changelog
		}
		existingChangelogs = existingMetadata.Changelogs
	}

	metadata := Metadata{
		Schema:     schema,
		Changelogs: existingChangelogs,
	}

	if newChangelog != nil {
		metadata.Changelogs = append(metadata.Changelogs, *newChangelog)
	}

	// Sort changelogs from newest to oldest
	sort.Slice(metadata.Changelogs, func(i, j int) bool {
		return metadata.Changelogs[i].Date.After(metadata.Changelogs[j].Date)
	})

	if err := writeMetadata(outputDir, metadata); err != nil {
		return nil, nil, fmt.Errorf("error writing holydocs data: %w", err)
	}

	return &metadata, newChangelog, nil
}

type outputDirectories struct {
	DiagramsDir           string
	ServiceDiagramDir     string
	MessageflowDiagramDir string
}

type diagramResults struct {
	OverviewDiagramPath string
	ServiceViews        []serviceView
	SystemDiagrams      map[string]systemDiagramView
	MessageFlowView     messageFlowView
}

func setupOutputDirectories(outputDir string) (*outputDirectories, error) {
	if err := os.MkdirAll(outputDir, dirPerm); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDirectoryCreationFailed, err)
	}

	diagramsDir := filepath.Join(outputDir, diagramsDirName)
	if err := os.RemoveAll(diagramsDir); err != nil {
		return nil, fmt.Errorf("failed to clean diagrams directory: %w", err)
	}

	if err := os.MkdirAll(diagramsDir, dirPerm); err != nil {
		return nil, fmt.Errorf("%w diagrams directory: %w", ErrDirectoryCreationFailed, err)
	}

	serviceDiagramDir := filepath.Join(diagramsDir, servicesDiagramDirName)
	if err := os.MkdirAll(serviceDiagramDir, dirPerm); err != nil {
		return nil, fmt.Errorf("%w service diagrams directory: %w", ErrDirectoryCreationFailed, err)
	}

	messageflowDiagramDir := filepath.Join(diagramsDir, messageflowDiagramDirName)
	if err := os.MkdirAll(messageflowDiagramDir, dirPerm); err != nil {
		return nil, fmt.Errorf("%w message flow diagrams directory: %w", ErrDirectoryCreationFailed, err)
	}

	return &outputDirectories{
		DiagramsDir:           diagramsDir,
		ServiceDiagramDir:     serviceDiagramDir,
		MessageflowDiagramDir: messageflowDiagramDir,
	}, nil
}

func generateAllDiagrams(
	ctx context.Context,
	schema domain.Schema,
	asyncEdges []asyncEdge,
	holydocsTarget domain.Target,
	messageflowSchema mf.Schema,
	messageflowTarget mf.Target,
	cfg *config.Config,
	outputDirs *outputDirectories,
) (*diagramResults, error) {
	overviewDiagramPath := filepath.Join(outputDirs.DiagramsDir, "overview.svg")
	if err := generateOverviewDiagram(ctx, schema, asyncEdges, holydocsTarget, cfg.Output.GlobalName,
		overviewDiagramPath, &cfg.Documentation); err != nil {
		return nil, fmt.Errorf("failed to generate overview diagram: %w", err)
	}

	serviceViews, err := buildServiceViews(ctx, schema, asyncEdges, holydocsTarget,
		messageflowSchema, messageflowTarget, outputDirs.ServiceDiagramDir, &cfg.Documentation)
	if err != nil {
		return nil, fmt.Errorf("failed to build service views: %w", err)
	}

	systemDiagrams, err := generateSystemDiagrams(ctx, schema, asyncEdges, holydocsTarget, outputDirs.DiagramsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to generate system diagrams: %w", err)
	}

	mfv, err := generateMessageFlowSection(ctx, messageflowSchema, messageflowTarget, outputDirs.MessageflowDiagramDir)
	if err != nil {
		return nil, fmt.Errorf("failed to generate message flow diagrams: %w", err)
	}

	return &diagramResults{
		OverviewDiagramPath: overviewDiagramPath,
		ServiceViews:        serviceViews,
		SystemDiagrams:      systemDiagrams,
		MessageFlowView:     mfv,
	}, nil
}

func buildTemplateData(
	cfg *config.Config,
	diagramResults *diagramResults,
	changelogs []domain.Changelog,
) templateData {
	overviewMarkdown := processMarkdown(cfg.Documentation.Overview.Description)

	serviceSummaries := make(map[string]string)
	for serviceName, serviceDoc := range cfg.Documentation.Services {
		serviceSummaries[serviceName] = processMarkdown(serviceDoc.Summary)
	}

	systemSummaries := make(map[string]string)
	systemMarkdowns := make(map[string]string)
	for systemName, systemDoc := range cfg.Documentation.Systems {
		systemSummaries[systemName] = processMarkdown(systemDoc.Summary)
		systemMarkdowns[systemName] = processMarkdown(systemDoc.Description)
	}

	return templateData{
		Title:           cfg.Output.Title,
		OverviewDiagram: filepath.ToSlash(filepath.Join(diagramsDirName, filepath.Base(diagramResults.OverviewDiagramPath))),
		OverviewD2: filepath.ToSlash(filepath.Join(diagramsDirName,
			strings.TrimSuffix(filepath.Base(diagramResults.OverviewDiagramPath), ".svg")+".d2")),
		OverviewMarkdown: overviewMarkdown,
		Systems:          groupServicesBySystem(diagramResults.ServiceViews),
		SystemDiagrams:   diagramResults.SystemDiagrams,
		SystemMarkdowns:  systemMarkdowns,
		ServiceSummaries: serviceSummaries,
		SystemSummaries:  systemSummaries,
		MessageFlow:      diagramResults.MessageFlowView,
		Changelogs:       changelogs,
	}
}

func processMarkdown(markdown config.Markdown) string {
	if markdown.Content != "" {
		return markdown.Content
	}

	if markdown.FilePath != "" {
		if content, err := os.ReadFile(markdown.FilePath); err == nil {
			return string(content)
		}
	}

	return ""
}

func generateSystemDiagrams(
	ctx context.Context,
	schema domain.Schema,
	asyncEdges []asyncEdge,
	target domain.Target,
	diagramsDir string,
) (map[string]systemDiagramView, error) {
	d2Target, ok := target.(*d2target.Target)
	if !ok {
		return nil, errors.New("target is not a D2 target")
	}

	systems := make(map[string]struct{})
	for _, service := range schema.Services {
		if systemName := strings.TrimSpace(service.Info.System); systemName != "" {
			systems[systemName] = struct{}{}
		}
	}

	systemDiagrams := make(map[string]systemDiagramView)

	for systemName := range systems {
		script, err := d2Target.GenerateSystemDiagramScript(schema, systemName, convertAsyncEdges(asyncEdges))
		if err != nil {
			return nil, fmt.Errorf("generate system D2 script for %s: %w", systemName, err)
		}

		if len(script) == 0 {
			continue
		}

		d2Filename := fmt.Sprintf("system-%s.d2", sanitizeFilename(systemName))
		d2Path := filepath.Join(diagramsDir, d2Filename)
		if err := os.WriteFile(d2Path, script, filePerm); err != nil {
			return nil, fmt.Errorf("write system D2 script for %s: %w", systemName, err)
		}

		diagram, err := d2Target.GenerateSystemDiagram(ctx, schema, systemName, convertAsyncEdges(asyncEdges))
		if err != nil {
			return nil, fmt.Errorf("render system diagram for %s: %w", systemName, err)
		}

		svgFilename := fmt.Sprintf("system-%s.svg", sanitizeFilename(systemName))
		svgPath := filepath.Join(diagramsDir, svgFilename)
		if err := os.WriteFile(svgPath, diagram, filePerm); err != nil {
			return nil, fmt.Errorf("write system diagram for %s: %w", systemName, err)
		}

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
	schema domain.Schema,
	asyncEdges []asyncEdge,
	holydocsTarget domain.Target,
	messageflowSchema mf.Schema,
	messageflowTarget mf.Target,
	outputDir string,
	documentation *DocumentationConfig,
) ([]serviceView, error) {
	serviceNameSet := buildServiceNameSet(schema.Services)
	edgesByService := buildEdgesByServiceMap(asyncEdges)

	views := make([]serviceView, 0, len(schema.Services))
	for _, service := range schema.Services {
		view, err := buildServiceView(ctx, service, schema.Services, edgesByService,
			holydocsTarget, messageflowSchema, messageflowTarget, serviceNameSet, outputDir, documentation)
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

func buildServiceNameSet(services []domain.Service) map[string]struct{} {
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
	service domain.Service,
	allServices []domain.Service,
	edgesByService map[string][]asyncEdge,
	holydocsTarget domain.Target,
	messageflowSchema mf.Schema,
	messageflowTarget mf.Target,
	serviceNameSet map[string]struct{},
	outputDir string,
	documentation *DocumentationConfig,
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

	// Use config summary if available, otherwise use servicefile description
	description := service.Info.Description
	if documentation != nil {
		if serviceDoc, exists := documentation.Services[service.Info.Name]; exists {
			if serviceDoc.Summary.Content != "" {
				description = serviceDoc.Summary.Content
			} else if serviceDoc.Summary.FilePath != "" {
				if content, err := os.ReadFile(serviceDoc.Summary.FilePath); err == nil {
					description = string(content)
				}
			}
		}
	}

	return serviceView{
		Name:        service.Info.Name,
		Anchor:      sanitizeAnchor(service.Info.Name),
		System:      service.Info.System,
		Description: d2target.FormatDescription(strings.TrimSpace(description)),
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
	holydocsTarget domain.Target, serviceNameSet map[string]struct{}) []asyncSummary {
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

func buildServiceFlowDiagram(
	ctx context.Context,
	service domain.Service,
	messageflowSchema mf.Schema,
	messageflowTarget mf.Target,
	outputDir,
	filenameBase string,
) string {
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

func buildRelationshipSummaries(rels []domain.Relationship) []relationshipSummary {
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

// modifySchemaWithServiceSummaries creates a modified schema with config-provided service summaries.
func modifySchemaWithServiceSummaries(schema domain.Schema, documentation *DocumentationConfig) domain.Schema {
	if documentation == nil {
		return schema
	}

	modifiedSchema := schema

	modifiedServices := make([]domain.Service, len(schema.Services))
	for i, service := range schema.Services {
		modifiedService := service

		if serviceDoc, exists := documentation.Services[service.Info.Name]; exists {
			modifiedServiceInfo := service.Info
			if serviceDoc.Summary.Content != "" {
				modifiedServiceInfo.Description = serviceDoc.Summary.Content
			} else if serviceDoc.Summary.FilePath != "" {
				if content, err := os.ReadFile(serviceDoc.Summary.FilePath); err == nil {
					modifiedServiceInfo.Description = string(content)
				}
			}
			modifiedService.Info = modifiedServiceInfo
		}

		modifiedServices[i] = modifiedService
	}

	modifiedSchema.Services = modifiedServices

	return modifiedSchema
}

// generateOverviewDiagramWithSystemContent creates a custom overview diagram that includes system content.
func generateOverviewDiagramWithSystemContent(
	d2Target *d2target.Target,
	schema domain.Schema,
	asyncEdges []domain.AsyncEdge,
	globalName string,
	documentation *DocumentationConfig,
) ([]byte, error) {
	// First, generate the standard overview diagram
	script, err := d2Target.GenerateOverviewDiagramScript(schema, asyncEdges, globalName)
	if err != nil {
		return nil, fmt.Errorf("generate standard overview D2 script: %w", err)
	}

	// Parse the generated script and modify system nodes to include content
	modifiedScript := modifySystemNodesInScript(string(script), schema, documentation)

	return []byte(modifiedScript), nil
}

// modifySystemNodesInScript modifies system nodes in the D2 script to include service summaries.
func modifySystemNodesInScript(script string, schema domain.Schema, documentation *DocumentationConfig) string {
	// Group services by system
	systemServices := make(map[string][]domain.Service)
	for _, service := range schema.Services {
		if systemName := strings.TrimSpace(service.Info.System); systemName != "" {
			systemServices[systemName] = append(systemServices[systemName], service)
		}
	}

	for systemName, services := range systemServices {
		systemNodeID := "internal.system_" + strings.ToLower(strings.ReplaceAll(systemName, " ", "-"))

		pattern := fmt.Sprintf("%s: |md\n# %s\n|", systemNodeID, systemName)
		replacement := systemNodeID + ": |md\n" + buildSystemDescription(systemName, services, documentation) + "\n|"

		script = strings.Replace(script, pattern, replacement, 1)
	}

	return script
}

// buildSystemDescription creates a description for a system that includes service summaries.
func buildSystemDescription(systemName string, _ []domain.Service, documentation *DocumentationConfig) string {
	var description strings.Builder

	description.WriteString(fmt.Sprintf("# %s\n\n", systemName))

	if systemDoc, exists := documentation.Systems[systemName]; exists {
		if systemDoc.Summary.Content != "" {
			summary := strings.TrimSpace(systemDoc.Summary.Content)
			description.WriteString(d2target.FormatOverviewDescription(summary))
		} else if systemDoc.Summary.FilePath != "" {
			if content, err := os.ReadFile(systemDoc.Summary.FilePath); err == nil {
				lines := strings.Split(string(content), "\n")
				var summaryLines []string
				for _, line := range lines {
					line = strings.TrimSpace(line)
					if line == "" || strings.HasPrefix(line, "#") {
						break // Stop at first empty line or header
					}
					summaryLines = append(summaryLines, line)
				}
				summary := strings.Join(summaryLines, " ")
				description.WriteString(d2target.FormatOverviewDescription(summary))
			}
		}
	}

	return description.String()
}

func generateOverviewDiagram(
	ctx context.Context,
	schema domain.Schema,
	asyncEdges []asyncEdge,
	target domain.Target,
	globalName, outputPath string,
	documentation *DocumentationConfig,
) error {
	d2Target, ok := target.(*d2target.Target)
	if !ok {
		return errors.New("target is not a D2 target")
	}

	modifiedSchema := modifySchemaWithServiceSummaries(schema, documentation)

	script, err := generateOverviewDiagramWithSystemContent(
		d2Target, modifiedSchema, convertAsyncEdges(asyncEdges), globalName, documentation)
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

	formatted := domain.FormattedSchema{
		Type: "d2",
		Data: script,
	}
	diagram, err := d2Target.RenderSchema(ctx, formatted)
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
	service domain.Service,
	allServices []domain.Service,
	serviceEdges []asyncEdge,
	target domain.Target,
	outputPath string,
) error {
	d2Target, ok := target.(*d2target.Target)
	if !ok {
		return errors.New("target is not a D2 target")
	}

	script, err := d2Target.GenerateServiceRelationshipsDiagramScript(service, allServices,
		convertAsyncEdges(serviceEdges))
	if err != nil {
		return fmt.Errorf("generate service relationships D2 script: %w", err)
	}

	if len(script) == 0 {
		return nil
	}

	d2Path := strings.TrimSuffix(outputPath, ".svg") + ".d2"
	if err := os.WriteFile(d2Path, script, filePerm); err != nil {
		return fmt.Errorf("write service relationships D2 script: %w", err)
	}

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

func readMetadata(outputDir string) (*Metadata, error) {
	metadataPath := filepath.Join(outputDir, "domain.json")

	if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
		return nil, nil // No existing metadata
	}

	data, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil, fmt.Errorf("error reading metadata file: %w", err)
	}

	var metadata Metadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("error unmarshaling metadata: %w", err)
	}

	return &metadata, nil
}

func writeMetadata(outputDir string, data Metadata) error {
	if err := os.MkdirAll(outputDir, dirPerm); err != nil {
		return fmt.Errorf("error creating output directory: %w", err)
	}

	metadataPath := filepath.Join(outputDir, "domain.json")

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling metadata: %w", err)
	}

	if err := os.WriteFile(metadataPath, jsonData, filePerm); err != nil {
		return fmt.Errorf("error writing metadata file: %w", err)
	}

	return nil
}
