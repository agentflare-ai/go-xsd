package xsd

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/agentflare-ai/go-xmldom"
)

// DocFormat identifies a documentation renderer.
type DocFormat string

const (
	// DocFormatMarkdown renders documentation as GitHub-flavored Markdown.
	DocFormatMarkdown DocFormat = "markdown"
	// DocFormatHTML is reserved for future HTML output.
	DocFormatHTML DocFormat = "html"
	// DocFormatAsciiDoc is reserved for future AsciiDoc output.
	DocFormatAsciiDoc DocFormat = "asciidoc"
	// DocFormatJSON is reserved for future machine-readable summaries.
	DocFormatJSON DocFormat = "json"
)

var supportedDocFormats = map[DocFormat]string{
	DocFormatMarkdown: "Markdown",
}

// SupportedDocFormats returns a copy of the supported format map.
func SupportedDocFormats() map[DocFormat]string {
	out := make(map[DocFormat]string, len(supportedDocFormats))
	for k, v := range supportedDocFormats {
		out[k] = v
	}
	return out
}

// DocOptions configures schema documentation generation.
type DocOptions struct {
	Title      string
	IncludeTOC bool
	Sections   []string
	MaxDepth   int
}

// GenerateDoc renders schema documentation using the requested format.
func GenerateDoc(schema *Schema, format DocFormat, opts DocOptions) (string, error) {
	if schema == nil {
		return "", fmt.Errorf("schema is nil")
	}

	if _, ok := supportedDocFormats[format]; !ok {
		return "", fmt.Errorf("doc format %s not supported yet", format)
	}

	if opts.MaxDepth <= 0 {
		opts.MaxDepth = 3
	}

	model := buildSchemaDocModel(schema, opts)

	switch format {
	case DocFormatMarkdown:
		return renderMarkdownDoc(model, opts), nil
	default:
		return "", fmt.Errorf("doc format %s not implemented", format)
	}
}

type schemaDocModel struct {
	Title            string
	SchemaDoc        string
	TargetNamespace  string
	ElementSummaries []elementDocModel
	TypeSummaries    []typeDocModel
	Constraints      []identityConstraintDocModel
	SectionFilter    map[string]struct{}
}

type elementDocModel struct {
	Name          QName
	Anchor        string
	TypeDisplay   string
	Cardinality   string
	Documentation string
	Attributes    []attributeDocModel
	Children      []childRef
	Wildcards     []string
	Constraints   []identityConstraintDocModel
	Nested        []elementDocModel
}

type childRef struct {
	Label       string
	Anchor      string
	Description string
	MinOcc      int
	MaxOcc      int
}

type attributeDocModel struct {
	Name        QName
	Use         AttributeUse
	TypeDisplay string
	Default     string
	Fixed       string
	Description string
}

type typeDocModel struct {
	Name           QName
	Anchor         string
	Kind           string // simple or complex
	BaseDisplay    string
	Documentation  string
	Facets         []facetDocModel
	Mixed          bool
	ContentSummary string
	Attributes     []attributeDocModel
}

type facetDocModel struct {
	Name  string
	Value string
}

type identityConstraintDocModel struct {
	Name     string
	Kind     IdentityConstraintKind
	Selector string
	Fields   []string
	Refer    QName
}

func buildSchemaDocModel(schema *Schema, opts DocOptions) schemaDocModel {
	schema.mu.RLock()
	elementDecls := make([]*ElementDecl, 0, len(schema.ElementDecls))
	for _, decl := range schema.ElementDecls {
		elementDecls = append(elementDecls, decl)
	}

	typeDefs := make([]Type, 0, len(schema.TypeDefs))
	for _, t := range schema.TypeDefs {
		typeDefs = append(typeDefs, t)
	}
	schema.mu.RUnlock()

	sort.SliceStable(elementDecls, func(i, j int) bool {
		return elementDecls[i].Name.String() < elementDecls[j].Name.String()
	})
	sort.SliceStable(typeDefs, func(i, j int) bool {
		return typeDefs[i].Name().String() < typeDefs[j].Name().String()
	})

	sectionFilter := make(map[string]struct{})
	for _, s := range opts.Sections {
		sectionFilter[strings.ToLower(strings.TrimSpace(s))] = struct{}{}
	}

	model := schemaDocModel{
		Title:           opts.Title,
		TargetNamespace: schema.TargetNamespace,
		SectionFilter:   sectionFilter,
		SchemaDoc:       extractSchemaDocumentation(schema),
	}

	anchors := make(map[string]string)
	for _, decl := range elementDecls {
		key := elementKey(decl.Name)
		anchors[key] = makeElementAnchor(decl.Name, "")
	}

	typeAnchors := make(map[string]string)
	for _, t := range typeDefs {
		typeAnchors[typeKey(t.Name())] = makeTypeAnchor(t.Name())
	}

	for _, decl := range elementDecls {
		model.ElementSummaries = append(model.ElementSummaries, describeElement(schema, decl, opts.MaxDepth, anchors, typeAnchors, ""))
	}

	for _, t := range typeDefs {
		model.TypeSummaries = append(model.TypeSummaries, describeType(schema, t, typeAnchors))
	}

	for _, decl := range elementDecls {
		for _, c := range decl.Constraints {
			model.Constraints = append(model.Constraints, identityConstraintDocModel{
				Name:     c.Name,
				Kind:     c.Kind,
				Selector: selectorXPath(c),
				Fields:   fieldsXPath(c),
				Refer:    c.Refer,
			})
		}
	}

	if model.Title == "" {
		model.Title = deriveSchemaTitle(schema.TargetNamespace)
	}

	return model
}

func describeElement(schema *Schema, decl *ElementDecl, maxDepth int, anchors map[string]string, typeAnchors map[string]string, overrideAnchor string) elementDocModel {
	cardinality := formatCardinality(decl.MinOcc, decl.MaxOcc)
	doc := elementDocModel{
		Name:        decl.Name,
		Cardinality: cardinality,
		TypeDisplay: "",
	}

	if overrideAnchor != "" {
		doc.Anchor = overrideAnchor
	} else {
		doc.Anchor = anchors[elementKey(decl.Name)]
		if doc.Anchor == "" {
			doc.Anchor = makeElementAnchor(decl.Name, "")
		}
	}

	doc.TypeDisplay = formatTypeRef(decl.Type, typeAnchors, schema.TargetNamespace)

	doc.Documentation = extractDocumentation(schema, "element", decl.Name)

	if ct, ok := decl.Type.(*ComplexType); ok {
		doc.Attributes = describeAttributes(schema, ct, typeAnchors)
		children, nested, wildcards := describeChildren(schema, ct.Content, anchors, typeAnchors, doc.Anchor, maxDepth-1)
		doc.Children = children
		doc.Wildcards = wildcards
		doc.Nested = nested
	}

	for _, constraint := range decl.Constraints {
		doc.Constraints = append(doc.Constraints, identityConstraintDocModel{
			Name:     constraint.Name,
			Kind:     constraint.Kind,
			Selector: selectorXPath(constraint),
			Fields:   fieldsXPath(constraint),
			Refer:    constraint.Refer,
		})
	}

	return doc
}

func describeType(schema *Schema, t Type, typeAnchors map[string]string) typeDocModel {
	doc := typeDocModel{
		Name:          t.Name(),
		Anchor:        typeAnchors[typeKey(t.Name())],
		Documentation: extractDocumentation(schema, typeLocalName(t), t.Name()),
	}

	switch st := t.(type) {
	case *SimpleType:
		doc.Kind = "simple"
		doc.BaseDisplay = formatQNameRef(st.Base, typeAnchors, schema.TargetNamespace)
		doc.Facets = describeFacets(st.Restriction)
		if st.List != nil {
			doc.ContentSummary = "list type"
		}
		if st.Union != nil {
			doc.ContentSummary = "union type"
		}
	case *ComplexType:
		doc.Kind = "complex"
		doc.Mixed = st.Mixed
		doc.ContentSummary = summarizeContent(st.Content)
		if ext := extractComplexBase(st.Content); ext != (QName{}) {
			doc.BaseDisplay = formatQNameRef(ext, typeAnchors, schema.TargetNamespace)
		}
		doc.Attributes = describeAttributes(schema, st, typeAnchors)
	default:
		doc.Kind = "complex"
	}

	return doc
}

func describeAttributes(schema *Schema, ct *ComplexType, typeAnchors map[string]string) []attributeDocModel {
	results := make([]attributeDocModel, 0, len(ct.Attributes))

	appendAttr := func(attr *AttributeDecl) {
		if attr == nil {
			return
		}
		results = append(results, attributeDocModel{
			Name:        attr.Name,
			Use:         attr.Use,
			TypeDisplay: formatTypeRef(attr.Type, typeAnchors, schema.TargetNamespace),
			Default:     attr.Default,
			Fixed:       attr.Fixed,
		})
	}

	for _, attr := range ct.Attributes {
		appendAttr(attr)
	}

	for _, groupName := range ct.AttributeGroup {
		if schema.AttributeGroups == nil {
			continue
		}
		if group, ok := schema.AttributeGroups[groupName]; ok && group != nil {
			for _, attr := range group.Attributes {
				appendAttr(attr)
			}
		}
	}

	sort.SliceStable(results, func(i, j int) bool {
		return results[i].Name.String() < results[j].Name.String()
	})
	return results
}

func describeFacets(rest *Restriction) []facetDocModel {
	if rest == nil {
		return nil
	}

	var facets []facetDocModel
	for _, facet := range rest.Facets {
		value := formatFacetValue(facet)
		facets = append(facets, facetDocModel{
			Name:  facet.Name(),
			Value: value,
		})
	}

	return facets
}

func formatFacetValue(facet FacetValidator) string {
	switch f := facet.(type) {
	case *EnumerationFacet:
		return strings.Join(f.Values, ", ")
	case *PatternFacet:
		return f.Pattern
	case *LengthFacet:
		return fmt.Sprintf("%d", f.Value)
	case *MinLengthFacet:
		return fmt.Sprintf("%d", f.Value)
	case *MaxLengthFacet:
		return fmt.Sprintf("%d", f.Value)
	case *MaxExclusiveFacet:
		return fmt.Sprintf("< %s", f.Value)
	case *MaxInclusiveFacet:
		return fmt.Sprintf("≤ %s", f.Value)
	case *MinExclusiveFacet:
		return fmt.Sprintf("> %s", f.Value)
	case *MinInclusiveFacet:
		return fmt.Sprintf("≥ %s", f.Value)
	case *FractionDigitsFacet:
		return fmt.Sprintf("%d", f.Value)
	case *TotalDigitsFacet:
		return fmt.Sprintf("%d", f.Value)
	default:
		return ""
	}
}

func selectorXPath(constraint *IdentityConstraint) string {
	if constraint == nil || constraint.Selector == nil {
		return ""
	}
	return constraint.Selector.XPath
}

func fieldsXPath(constraint *IdentityConstraint) []string {
	if constraint == nil {
		return nil
	}
	fields := make([]string, 0, len(constraint.Fields))
	for _, field := range constraint.Fields {
		if field != nil {
			fields = append(fields, field.XPath)
		}
	}
	return fields
}

func extractDocumentation(schema *Schema, kind string, name QName) string {
	if schema == nil || schema.doc == nil {
		return ""
	}

	root := schema.doc.DocumentElement()
	if root == nil {
		return ""
	}

	children := root.Children()
	for i := uint(0); i < children.Length(); i++ {
		child := children.Item(i)
		if child == nil {
			continue
		}
		if string(child.LocalName()) == kind {
			if n := child.GetAttribute("name"); string(n) == name.Local {
				if doc := extractAnnotation(child); doc != "" {
					return doc
				}
			}
		}
	}

	return ""
}

func extractAnnotation(elem xmldom.Element) string {
	children := elem.Children()
	for i := uint(0); i < children.Length(); i++ {
		child := children.Item(i)
		if child == nil {
			continue
		}
		if string(child.LocalName()) == "annotation" {
			grand := child.Children()
			for j := uint(0); j < grand.Length(); j++ {
				docNode := grand.Item(j)
				if docNode != nil && string(docNode.LocalName()) == "documentation" {
					return strings.TrimSpace(string(docNode.TextContent()))
				}
			}
		}
	}
	return ""
}

func formatCardinality(min, max int) string {
	switch {
	case min == 0 && max == 1:
		return "optional"
	case min == 0 && max < 0:
		return "0..∞"
	case min == 1 && max < 0:
		return "1..∞"
	case max >= 0 && min == max:
		return fmt.Sprintf("exactly %d", min)
	case max >= 0:
		return fmt.Sprintf("%d..%d", min, max)
	default:
		return fmt.Sprintf("%d..∞", min)
	}
}

func formatCardinalityProse(min, max int) string {
	switch {
	case min == 0 && max == 1:
		return "0 or 1 times"
	case min == 1 && max == 1:
		return "1 time"
	case min == 0 && max < 0:
		return "0 or more times"
	case min == 1 && max < 0:
		return "1 or more times"
	case max >= 0 && min == max:
		return fmt.Sprintf("exactly %d times", min)
	case max >= 0:
		return fmt.Sprintf("%d to %d times", min, max)
	default:
		return fmt.Sprintf("%d or more times", min)
	}
}

func describeChildren(schema *Schema, content Content, anchors map[string]string, typeAnchors map[string]string, parentAnchor string, depth int) ([]childRef, []elementDocModel, []string) {
	if depth < 0 || content == nil {
		return nil, nil, nil
	}

	var children []childRef
	var nested []elementDocModel
	wildcards := make([]string, 0)

	var walkContent func(Content)
	var walkParticle func(Particle)

	walkContent = func(content Content) {
		switch c := content.(type) {
		case *ModelGroup:
			for _, particle := range c.Particles {
				walkParticle(particle)
			}
		case *ComplexContent:
			if c.Extension != nil && c.Extension.Content != nil {
				walkContent(c.Extension.Content)
			}
			if c.Restriction != nil && c.Restriction.Content != nil {
				walkContent(c.Restriction.Content)
			}
		case *AllowAnyContent:
			wildcards = append(wildcards, "##any")
		}
	}

	walkParticle = func(particle Particle) {
		if particle == nil {
			return
		}
		switch p := particle.(type) {
		case *ElementDecl:
			var anchor string
			key := elementKey(p.Name)
			if existing, ok := anchors[key]; ok {
				anchor = existing
			} else {
				anchor = buildChildAnchor(parentAnchor, p.Name)
				nestedModel := describeElement(schema, p, depth, anchors, typeAnchors, anchor)
				nested = append(nested, nestedModel)
			}
			children = append(children, childRef{
				Label:       p.Name.Local,
				Anchor:      anchor,
				MinOcc:      p.MinOcc,
				MaxOcc:      p.MaxOcc,
				Description: extractDocumentation(schema, "element", p.Name),
			})
		case *ElementRef:
			schema.mu.RLock()
			decl := schema.ElementDecls[p.Ref]
			schema.mu.RUnlock()
			var anchor string
			var desc string
			if decl != nil {
				anchor = anchors[elementKey(decl.Name)]
				desc = extractDocumentation(schema, "element", decl.Name)
			}
			children = append(children, childRef{
				Label:       p.Ref.Local,
				Anchor:      anchor,
				MinOcc:      p.MinOcc,
				MaxOcc:      p.MaxOcc,
				Description: desc,
			})
		case *GroupRef:
			schema.mu.RLock()
			group := schema.Groups[p.Ref]
			schema.mu.RUnlock()
			if group != nil {
				for _, child := range group.Particles {
					walkParticle(child)
				}
			}
		case *AnyElement:
			ns := p.Namespace
			if ns == "" {
				ns = "##any"
			}
			wildcards = append(wildcards, ns)
		case *ModelGroup:
			for _, child := range p.Particles {
				walkParticle(child)
			}
		}
	}

	walkContent(content)

	seen := make(map[string]bool)
	deduped := make([]childRef, 0, len(children))
	for _, child := range children {
		key := child.Label + "|" + child.Anchor
		if !seen[key] {
			seen[key] = true
			deduped = append(deduped, child)
		}
	}

	wildcards = dedupeStrings(wildcards)

	return deduped, nested, wildcards
}

func summarizeContent(content Content) string {
	switch c := content.(type) {
	case *ModelGroup:
		return fmt.Sprintf("%s (%d particle(s))", c.Kind, len(c.Particles))
	case *SimpleContent:
		return fmt.Sprintf("simpleContent base=%s", c.Base.String())
	case *ComplexContent:
		if c.Extension != nil {
			return fmt.Sprintf("complexContent extends %s", c.Extension.Base.String())
		}
		if c.Restriction != nil {
			return fmt.Sprintf("complexContent restricts %s", c.Restriction.Base.String())
		}
	case *AllowAnyContent:
		return "any child elements permitted"
	case *AnyElement:
		return fmt.Sprintf("xs:any (%s)", c.Namespace)
	}
	return ""
}

func extractComplexBase(content Content) QName {
	if cc, ok := content.(*ComplexContent); ok {
		if cc.Extension != nil {
			return cc.Extension.Base
		}
		if cc.Restriction != nil {
			return cc.Restriction.Base
		}
	}
	return QName{}
}

func typeLocalName(t Type) string {
	switch t.(type) {
	case *SimpleType:
		return "simpleType"
	case *ComplexType:
		return "complexType"
	default:
		return "type"
	}
}

func elementKey(q QName) string {
	return q.Namespace + "|" + q.Local
}

func typeKey(q QName) string {
	return q.Namespace + "|" + q.Local
}

func makeElementAnchor(name QName, parent string) string {
	base := sanitizeAnchorPart(name.Namespace) + "-" + sanitizeAnchorPart(name.Local)
	if base == "-" {
		base = sanitizeAnchorPart(name.Local)
	}
	if parent != "" {
		return parent + "-" + sanitizeAnchorPart(name.Local)
	}
	return "element-" + strings.Trim(base, "-")
}

func makeTypeAnchor(name QName) string {
	base := sanitizeAnchorPart(name.Namespace) + "-" + sanitizeAnchorPart(name.Local)
	if base == "-" {
		base = sanitizeAnchorPart(name.Local)
	}
	return "type-" + strings.Trim(base, "-")
}

func buildChildAnchor(parent string, name QName) string {
	if parent == "" {
		return makeElementAnchor(name, "")
	}
	return parent + "-" + sanitizeAnchorPart(name.Local)
}

func sanitizeAnchorPart(s string) string {
	if s == "" {
		return "-"
	}
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteRune('-')
		}
	}
	result := strings.Trim(b.String(), "-")
	if result == "" {
		return "-"
	}
	return result
}

func formatTypeRef(t Type, typeAnchors map[string]string, targetNamespace string) string {
	if t == nil {
		return ""
	}
	return formatQNameRef(t.Name(), typeAnchors, targetNamespace)
}

func formatQNameRef(q QName, typeAnchors map[string]string, targetNamespace string) string {
	if q.Local == "" {
		return ""
	}
	if anchor := typeAnchors[typeKey(q)]; anchor != "" {
		return fmt.Sprintf("[%s](#%s)", q.Local, anchor)
	}
	if q.Namespace == "" || q.Namespace == targetNamespace {
		return fmt.Sprintf("`%s`", q.Local)
	}
	if q.Namespace == XSDNamespace {
		return fmt.Sprintf("`xs:%s`", q.Local)
	}
	return fmt.Sprintf("`%s` (%s)", q.Local, q.Namespace)
}

func formatNamespaceDisplay(ns string) string {
	if isURL(ns) {
		return fmt.Sprintf("[%s](%s)", ns, ns)
	}
	return fmt.Sprintf("`%s`", ns)
}

func isURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

func deriveSchemaTitle(ns string) string {
	ns = strings.TrimSpace(ns)
	if ns == "" {
		return "Schema Documentation"
	}
	trimmed := strings.TrimSuffix(ns, "/")
	parts := strings.Split(trimmed, "/")
	last := parts[len(parts)-1]
	if last == "" && len(parts) > 1 {
		last = parts[len(parts)-2]
	}
	if last == "" {
		return ns
	}
	last = strings.TrimSpace(last)
	if last == "" {
		return ns
	}
	title := strings.ToUpper(last[:1]) + last[1:]
	return fmt.Sprintf("%s Schema", title)
}

func escapeTag(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	return fmt.Sprintf("&lt;%s&gt;", name)
}

func compactText(s string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(s)), " ")
}

func composeNumber(base string, idx int) string {
	if base == "" {
		return strconv.Itoa(idx)
	}
	return base + "." + strconv.Itoa(idx)
}

func dedupeStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, v := range values {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func assignElementNumbers(elems []elementDocModel, base string, numbers map[string]string) {
	for i := range elems {
		number := composeNumber(base, i+1)
		if elems[i].Anchor != "" {
			numbers[elems[i].Anchor] = number
		}
		if len(elems[i].Nested) > 0 {
			assignElementNumbers(elems[i].Nested, number, numbers)
		}
	}
}

func assignTypeNumbers(types []typeDocModel, base string, numbers map[string]string) {
	for i := range types {
		number := composeNumber(base, i+1)
		if types[i].Anchor != "" {
			numbers[types[i].Anchor] = number
		}
	}
}

func extractSchemaDocumentation(schema *Schema) string {
	if schema == nil || schema.doc == nil {
		return ""
	}
	root := schema.doc.DocumentElement()
	if root == nil {
		return ""
	}
	return compactText(extractAnnotation(root))
}

func formatChildLine(child childRef, elementNumbers map[string]string) string {
	name := child.Label
	if name == "" {
		name = "<unknown>"
	}
	display := escapeTag(name)
	desc := compactText(child.Description)
	if desc == "" {
		desc = "Child element."
	}
	occurs := formatCardinalityProse(child.MinOcc, child.MaxOcc)
	line := fmt.Sprintf("%s %s Occurs %s.", display, desc, occurs)
	targetNum := elementNumbers[child.Anchor]
	if targetNum != "" && child.Anchor != "" {
		line += fmt.Sprintf(" See [%s %s](#%s).", targetNum, display, child.Anchor)
	} else if child.Anchor != "" {
		line += fmt.Sprintf(" See [%s](#%s).", display, child.Anchor)
	}
	return line
}

func writeElementTOC(b *strings.Builder, elems []elementDocModel, numbers map[string]string, depth int) {
	indent := strings.Repeat("  ", depth)
	for _, elem := range elems {
		name := elem.Name.Local
		if name == "" {
			name = elem.Name.String()
		}
		if name == "" {
			name = "(anonymous element)"
		}
		number := numbers[elem.Anchor]
		label := escapeTag(name)
		if number != "" {
			label = fmt.Sprintf("%s %s", number, label)
		}
		if elem.Anchor != "" {
			b.WriteString(fmt.Sprintf("%s- [%s](#%s)\n", indent, label, elem.Anchor))
		} else {
			b.WriteString(fmt.Sprintf("%s- %s\n", indent, label))
		}
		if len(elem.Nested) > 0 {
			writeElementTOC(b, elem.Nested, numbers, depth+1)
		}
	}
}

func writeTypeTOC(b *strings.Builder, types []typeDocModel, numbers map[string]string, depth int) {
	indent := strings.Repeat("  ", depth)
	for _, typ := range types {
		name := typ.Name.Local
		if name == "" {
			name = typ.Name.String()
		}
		if name == "" {
			name = "(anonymous type)"
		}
		number := numbers[typ.Anchor]
		label := name
		if number != "" {
			label = number + " " + name
		}
		if typ.Anchor != "" {
			b.WriteString(fmt.Sprintf("%s- [%s](#%s)\n", indent, label, typ.Anchor))
		} else {
			b.WriteString(fmt.Sprintf("%s- %s\n", indent, label))
		}
	}
}

func renderMarkdownDoc(model schemaDocModel, opts DocOptions) string {
	var b strings.Builder
	title := model.Title
	if isURL(model.Title) {
		title = fmt.Sprintf("[%s](%s)", model.Title, model.Title)
	}
	b.WriteString("# ")
	b.WriteString(title)
	b.WriteString("\n\n")

	if model.SchemaDoc != "" {
		b.WriteString(model.SchemaDoc)
		b.WriteString("\n\n")
	}

	include := func(section string) bool {
		if len(model.SectionFilter) == 0 {
			return true
		}
		_, ok := model.SectionFilter[section]
		return ok
	}

	// Assign section numbers
	section := 1
	var overviewNumber, elementsNumber, typesNumber, constraintsNumber string
	if include("overview") {
		overviewNumber = strconv.Itoa(section)
		section++
	}
	if include("elements") {
		elementsNumber = strconv.Itoa(section)
		section++
	}
	if include("types") {
		typesNumber = strconv.Itoa(section)
		section++
	}
	if include("constraints") && len(model.Constraints) > 0 {
		constraintsNumber = strconv.Itoa(section)
		section++
	}

	elementNumbers := make(map[string]string)
	if elementsNumber != "" {
		assignElementNumbers(model.ElementSummaries, elementsNumber, elementNumbers)
	}
	typeNumbers := make(map[string]string)
	if typesNumber != "" {
		assignTypeNumbers(model.TypeSummaries, typesNumber, typeNumbers)
	}

	if opts.IncludeTOC {
		b.WriteString("## Contents\n")
		if include("overview") {
			b.WriteString(fmt.Sprintf("- [%s Overview](#overview)\n", overviewNumber))
		}
		if include("elements") {
			b.WriteString(fmt.Sprintf("- [%s Elements](#elements)\n", elementsNumber))
			writeElementTOC(&b, model.ElementSummaries, elementNumbers, 1)
		}
		if include("types") {
			b.WriteString(fmt.Sprintf("- [%s Types](#types)\n", typesNumber))
			writeTypeTOC(&b, model.TypeSummaries, typeNumbers, 1)
		}
		if include("constraints") {
			b.WriteString(fmt.Sprintf("- [%s Identity Constraints](#identity-constraints)\n", constraintsNumber))
		}
		b.WriteString("\n")
	}

	if include("overview") {
		b.WriteString(fmt.Sprintf("## %s Overview\n\n", overviewNumber))
		if model.TargetNamespace != "" {
			b.WriteString(fmt.Sprintf("- **Target namespace:** %s\n", formatNamespaceDisplay(model.TargetNamespace)))
		}
		b.WriteString(fmt.Sprintf("- **Elements:** %d\n", len(model.ElementSummaries)))
		b.WriteString(fmt.Sprintf("- **Types:** %d\n", len(model.TypeSummaries)))
		b.WriteString("\n")
	}

	if include("elements") {
		b.WriteString(fmt.Sprintf("## %s Elements\n\n", elementsNumber))
		for _, elem := range model.ElementSummaries {
			writeElementSection(&b, elem, 3, elementNumbers[elem.Anchor], elementNumbers)
		}
	}

	if include("types") {
		b.WriteString(fmt.Sprintf("## %s Types\n\n", typesNumber))
		for _, typ := range model.TypeSummaries {
			writeTypeMarkdown(&b, typ, typeNumbers[typ.Anchor])
		}
	}

	if include("constraints") && len(model.Constraints) > 0 {
		b.WriteString(fmt.Sprintf("## %s Identity Constraints\n\n", constraintsNumber))
		for i, constraint := range model.Constraints {
			number := composeNumber(constraintsNumber, i+1)
			b.WriteString(fmt.Sprintf("### %s %s (%s)\n", number, constraint.Name, constraint.Kind))
			if constraint.Selector != "" {
				b.WriteString(fmt.Sprintf("- **Selector:** `%s`\n", constraint.Selector))
			}
			if len(constraint.Fields) > 0 {
				b.WriteString(fmt.Sprintf("- **Fields:** `%s`\n", strings.Join(constraint.Fields, "`, `")))
			}
			if constraint.Refer.Local != "" {
				b.WriteString(fmt.Sprintf("- **Refers to:** `%s`\n", constraint.Refer.String()))
			}
			b.WriteString("\n")
		}
	}

	return b.String()
}

func writeElementSection(b *strings.Builder, elem elementDocModel, level int, number string, elementNumbers map[string]string) {
	if elem.Anchor != "" {
		b.WriteString(fmt.Sprintf("<a id=\"%s\"></a>\n", elem.Anchor))
	}

	title := elem.Name.Local
	if title == "" {
		title = elem.Name.String()
	}
	if title == "" {
		title = "(anonymous element)"
	}
	if number != "" {
		title = fmt.Sprintf("%s %s", number, escapeTag(title))
	} else {
		title = escapeTag(title)
	}

	if level < 2 {
		level = 2
	}
	if level > 6 {
		level = 6
	}
	b.WriteString(fmt.Sprintf("%s %s\n\n", strings.Repeat("#", level), title))

	var detailLines []string
	if elem.Documentation != "" {
		b.WriteString("**Description**\n```\n")
		b.WriteString(elem.Documentation)
		b.WriteString("\n```\n\n")
	}
	if elem.Name.Namespace != "" {
		detailLines = append(detailLines, fmt.Sprintf("- **Namespace:** %s", elem.Name.Namespace))
	}
	if elem.TypeDisplay != "" {
		detailLines = append(detailLines, fmt.Sprintf("- **Type:** %s", elem.TypeDisplay))
	}
	if elem.Cardinality != "" {
		detailLines = append(detailLines, fmt.Sprintf("- **Cardinality:** %s", elem.Cardinality))
	}
	if len(elem.Children) > 0 {
	}
	if len(elem.Wildcards) > 0 {
		detailLines = append(detailLines, fmt.Sprintf("- **Extensibility:** allows any elements (namespace: %s)", strings.Join(elem.Wildcards, ", ")))
	}
	if len(elem.Constraints) > 0 {
		names := make([]string, len(elem.Constraints))
		for i, c := range elem.Constraints {
			names[i] = fmt.Sprintf("%s (%s)", c.Name, c.Kind)
		}
		detailLines = append(detailLines, fmt.Sprintf("- **Constraints:** %s", strings.Join(names, ", ")))
	}

	for _, line := range detailLines {
		b.WriteString(line)
		b.WriteString("\n")
	}

	if len(detailLines) > 0 {
		b.WriteString("\n")
	}

	if len(elem.Attributes) > 0 {
		b.WriteString("| Attribute | Use | Type | Default | Fixed |\n")
		b.WriteString("|-----------|-----|------|---------|-------|\n")
		for _, attr := range elem.Attributes {
			def := attr.Default
			if def == "" {
				def = "—"
			}
			fixed := attr.Fixed
			if fixed == "" {
				fixed = "—"
			}
			typeText := attr.TypeDisplay
			if typeText == "" {
				typeText = "—"
			}
			b.WriteString(fmt.Sprintf("| `%s` | %s | %s | %s | %s |\n",
				attr.Name.String(), attr.Use, typeText, def, fixed))
		}
	}

	b.WriteString("\n")

	if len(elem.Children) > 0 {
		childHeadingNumber := composeNumber(number, 2)
		if childHeadingNumber == "" {
			childHeadingNumber = "Children"
		}
		childHeader := fmt.Sprintf("%s %s Children\n\n", strings.Repeat("#", level+1), childHeadingNumber)
		b.WriteString(childHeader)
		for _, child := range elem.Children {
			b.WriteString(formatChildLine(child, elementNumbers))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	if len(elem.Nested) > 0 {
		for _, nested := range elem.Nested {
			writeElementSection(b, nested, level+1, elementNumbers[nested.Anchor], elementNumbers)
		}
	}
}

func writeTypeMarkdown(b *strings.Builder, typ typeDocModel, number string) {
	if typ.Anchor != "" {
		b.WriteString(fmt.Sprintf("<a id=\"%s\"></a>\n", typ.Anchor))
	}

	title := typ.Name.Local
	if title == "" {
		title = typ.Name.String()
	}
	if title == "" {
		title = "(anonymous type)"
	}
	if number != "" {
		title = number + " " + title
	}
	b.WriteString(fmt.Sprintf("### %s\n\n", title))

	var detailLines []string
	detailLines = append(detailLines, fmt.Sprintf("- **Kind:** %s", typ.Kind))
	if typ.Name.Namespace != "" {
		detailLines = append(detailLines, fmt.Sprintf("- **Namespace:** %s", typ.Name.Namespace))
	}
	if typ.BaseDisplay != "" {
		detailLines = append(detailLines, fmt.Sprintf("- **Base:** %s", typ.BaseDisplay))
	}
	if typ.Mixed {
		detailLines = append(detailLines, "- **Mixed content:** yes")
	}
	if typ.ContentSummary != "" {
		detailLines = append(detailLines, fmt.Sprintf("- **Content:** %s", typ.ContentSummary))
	}

	if typ.Documentation != "" {
		b.WriteString("**Description**\n```\n")
		b.WriteString(typ.Documentation)
		b.WriteString("\n```\n\n")
	}

	for _, line := range detailLines {
		b.WriteString(line)
		b.WriteString("\n")
	}

	if len(detailLines) > 0 {
		b.WriteString("\n")
	}

	if len(typ.Facets) > 0 {
		b.WriteString("| Facet | Value |\n")
		b.WriteString("|-------|-------|\n")
		for _, facet := range typ.Facets {
			b.WriteString(fmt.Sprintf("| %s | %s |\n", facet.Name, facet.Value))
		}
	}

	if len(typ.Attributes) > 0 {
		b.WriteString("\n| Attribute | Use | Type | Default | Fixed |\n")
		b.WriteString("|-----------|-----|------|---------|-------|\n")
		for _, attr := range typ.Attributes {
			def := attr.Default
			if def == "" {
				def = "—"
			}
			fixed := attr.Fixed
			if fixed == "" {
				fixed = "—"
			}
			typeText := attr.TypeDisplay
			if typeText == "" {
				typeText = "—"
			}
			b.WriteString(fmt.Sprintf("| `%s` | %s | %s | %s | %s |\n",
				attr.Name.String(), attr.Use, typeText, def, fixed))
		}
	}

	b.WriteString("\n")
}
