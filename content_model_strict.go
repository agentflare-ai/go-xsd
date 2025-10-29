package xsd

import (
	"github.com/agentflare-ai/go-xmldom"
)

type intSet map[int]struct{}

func (s intSet) add(i int) { s[i] = struct{}{} }

// contentMatcher implements a strict content-model validator using backtracking with memoization
// It handles sequence and choice robustly, and delegates 'all' to the existing validator/count helpers.
// It focuses on structure acceptance; datatype and wildcard processContents are validated elsewhere.
type contentMatcher struct {
	schema *Schema
	// Memoization for one-occurrence group matching: key=(group pointer, idx)
	memoGroup map[*ModelGroup]map[int]intSet
	// Memoization for sequence progress: key by group pointer and (particle index, idx)
	memoSeq map[*ModelGroup]map[[2]int]intSet
}

func newContentMatcher(schema *Schema) *contentMatcher {
	return &contentMatcher{
		schema:    schema,
		memoGroup: make(map[*ModelGroup]map[int]intSet),
		memoSeq:   make(map[*ModelGroup]map[[2]int]intSet),
	}
}

// matchComplexType returns true if children structurally match the complex type content model
func (cm *contentMatcher) matchComplexType(ct *ComplexType, children []xmldom.Element) bool {
	if ct == nil {
		return len(children) == 0
	}
	if ct.Content == nil {
		return len(children) == 0
	}
	// Extract an underlying model group to match
	switch c := ct.Content.(type) {
	case *ModelGroup:
		return cm.matchAllChildren(c, children)
	case *ComplexContent:
		var inner Content
		if c.Extension != nil {
			inner = c.Extension.Content
		}
		if c.Restriction != nil {
			inner = c.Restriction.Content
		}
		if mg, ok := inner.(*ModelGroup); ok {
			return cm.matchAllChildren(mg, children)
		}
		return len(children) == 0
	case *SimpleContent:
		// No element children permitted
		return len(children) == 0
	default:
		return len(children) == 0
	}
}

func (cm *contentMatcher) matchAllChildren(mg *ModelGroup, children []xmldom.Element) bool {
	// Special-case: choice with a single wildcard particle applies to the entire child list
	if mg.Kind == ChoiceGroup && len(mg.Particles) == 1 {
		if wc, ok := mg.Particles[0].(*AnyElement); ok {
			count := 0
			for _, ch := range children {
				if !MatchesWildcard(ch, wc.Namespace, cm.schema.TargetNamespace) {
					return false // contains element not permitted by wildcard
				}
				count++
			}
			if count < wc.MinOcc {
				return false
			}
			if wc.MaxOcc != -1 && count > wc.MaxOcc {
				return false
			}
			return true
		}
	}
	out := cm.matchGroupRepeat(mg, children, 0, mg.MinOcc, mg.MaxOcc)
	// Accept if any end index equals len(children)
	if out == nil {
		return false
	}
	_, ok := out[len(children)]
	return ok
}

// matchGroupRepeat applies occurrences for a particle that is a model group
func (cm *contentMatcher) matchGroupRepeat(mg *ModelGroup, children []xmldom.Element, idx, min, max int) intSet {
	// Start with zero occurrences
	result := make(intSet, 4)
	if min == 0 {
		result.add(idx)
	}
	// Accumulate by applying one-occurrence matches repeatedly
	current := make(intSet, 4)
	current.add(idx)
	occ := 0
	for max == -1 || occ < max {
		nextIndices := make(intSet, 8)
		zeroProgress := true
		for i := range current {
			outs := cm.matchGroupOnce(mg, children, i)
			for j := range outs {
				nextIndices.add(j)
				if j != i {
					zeroProgress = false
				}
			}
		}
		if len(nextIndices) == 0 || zeroProgress {
			break
		}
		occ++
		current = nextIndices
		if occ >= min {
			for i := range current {
				result.add(i)
			}
		}
	}
	return result
}

// matchGroupOnce returns possible indices after matching exactly one occurrence of mg starting at idx
func (cm *contentMatcher) matchGroupOnce(mg *ModelGroup, children []xmldom.Element, idx int) intSet {
	if idx > len(children) {
		return nil
	}
	// Memoize per (mg, idx)
	if m, ok := cm.memoGroup[mg]; ok {
		if s, ok2 := m[idx]; ok2 {
			return s
		}
	} else {
		cm.memoGroup[mg] = make(map[int]intSet)
	}
	var out intSet
	switch mg.Kind {
	case SequenceGroup:
		out = cm.matchSequenceOnce(mg, children, idx)
	case ChoiceGroup:
		out = cm.matchChoiceOnce(mg, children, idx)
	case AllGroup:
		// Delegate to existing validator for one occurrence check and consumed length
		violations := mg.validateAllStrict(children[idx:], cm.schema)
		if len(violations) == 0 {
			consumed := mg.countConsumedByGroup(mg, children[idx:], cm.schema)
			if consumed == 0 {
				// zero-length match permitted for 'all'
				out = make(intSet, 1)
				out.add(idx)
			} else {
				out = make(intSet, 1)
				out.add(idx + consumed)
			}
		} else {
			out = make(intSet) // empty
		}
	}
	cm.memoGroup[mg][idx] = out
	return out
}

func (cm *contentMatcher) matchSequenceOnce(mg *ModelGroup, children []xmldom.Element, idx int) intSet {
	// seqKey: particle index, idx
	if cm.memoSeq[mg] == nil {
		cm.memoSeq[mg] = make(map[[2]int]intSet)
	}
	var rec func(pi, i int) intSet
	rec = func(pi, i int) intSet {
		key := [2]int{pi, i}
		if s, ok := cm.memoSeq[mg][key]; ok {
			return s
		}
		// End of particles -> success at current index
		if pi >= len(mg.Particles) {
			s := make(intSet, 1)
			s.add(i)
			cm.memoSeq[mg][key] = s
			return s
		}
		p := mg.Particles[pi]
		// Expand occurrences for this particle
		min := p.MinOccurs()
		max := p.MaxOccurs()
		// For zero occurrences allowed, we can immediately try next particle
		result := make(intSet, 8)
		if min == 0 {
			for j := range rec(pi+1, i) {
				result.add(j)
			}
		}
		// Apply one occurrence repeatedly
		frontier := make(intSet, 4)
		frontier.add(i)
		occ := 0
		for max == -1 || occ < max {
			nextF := make(intSet, 8)
			progress := false
			for pos := range frontier {
				outs := cm.matchParticleOnce(mg, p, children, pos)
				for j := range outs {
					nextF.add(j)
					if j != pos {
						progress = true
					}
				}
			}
			if len(nextF) == 0 || !progress {
				break
			}
			occ++
			frontier = nextF
			if occ >= min {
				// After satisfying min, continue with next particle
				for pos := range frontier {
					for j := range rec(pi+1, pos) {
						result.add(j)
					}
				}
			}
		}
		cm.memoSeq[mg][key] = result
		return result
	}
	return rec(0, idx)
}

func (cm *contentMatcher) matchChoiceOnce(mg *ModelGroup, children []xmldom.Element, idx int) intSet {
	result := make(intSet, 8)
	for _, alt := range mg.Particles {
		// Repeat alternative per its occurrence
		for j := range cm.matchParticleRepeat(mg, alt, children, idx, alt.MinOccurs(), alt.MaxOccurs()) {
			result.add(j)
		}
	}
	return result
}

func (cm *contentMatcher) matchParticleRepeat(parent *ModelGroup, p Particle, children []xmldom.Element, idx, min, max int) intSet {
	res := make(intSet, 4)
	if min == 0 {
		res.add(idx)
	}
	frontier := make(intSet, 4)
	frontier.add(idx)
	occ := 0
	for max == -1 || occ < max {
		nextF := make(intSet, 8)
		progress := false
		for pos := range frontier {
			outs := cm.matchParticleOnce(parent, p, children, pos)
			for j := range outs {
				nextF.add(j)
				if j != pos {
					progress = true
				}
			}
		}
		if len(nextF) == 0 || !progress {
			break
		}
		occ++
		frontier = nextF
		if occ >= min {
			for j := range frontier {
				res.add(j)
			}
		}
	}
	return res
}

// matchParticleOnce tries to match exactly one occurrence of p at index idx
func (cm *contentMatcher) matchParticleOnce(parent *ModelGroup, p Particle, children []xmldom.Element, idx int) intSet {
	out := make(intSet, 2)
	if idx >= len(children) {
		return out
	}
	switch t := p.(type) {
	case *ElementDecl:
		ch := children[idx]
		q := QName{Namespace: string(ch.NamespaceURI()), Local: string(ch.LocalName())}
		if q == t.Name || cm.schema.isSubstitutableFor(q, t.Name) {
			out.add(idx + 1)
		}
	case *ElementRef:
		ch := children[idx]
		q := QName{Namespace: string(ch.NamespaceURI()), Local: string(ch.LocalName())}
		if q == t.Ref || cm.schema.isSubstitutableFor(q, t.Ref) {
			out.add(idx + 1)
		}
	case *AnyElement:
		if MatchesWildcard(children[idx], t.Namespace, cm.schema.TargetNamespace) {
			out.add(idx + 1)
		}
	case *GroupRef:
		cm.schema.mu.RLock()
		g := cm.schema.Groups[t.Ref]
		cm.schema.mu.RUnlock()
		if g != nil {
			outs := cm.matchGroupOnce(g, children, idx)
			for j := range outs {
				out.add(j)
			}
		}
	case *ModelGroup:
		outs := cm.matchGroupOnce(t, children, idx)
		for j := range outs {
			out.add(j)
		}
	default:
		// Unknown particle type -> no match
	}
	return out
}
