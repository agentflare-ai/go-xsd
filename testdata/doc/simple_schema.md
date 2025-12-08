# Agentml Schema

AgentML 1.0 core schema for the `github.com/agentflare-ai/agentml` namespace. Defines the AgentML state machine language used to express autonomous agents as hierarchical state machines compatible with SCXML, extended with executable content, datamodel operations, JSON Schema–driven validation, and external communication primitives.

## Contents
- [1 Overview](#overview)
- [2 Elements](#elements)
  - [2.1 &lt;agent&gt;](#element-github-com-agentflare-ai-agentml-agent)
  - [2.2 &lt;agentml&gt;](#element-github-com-agentflare-ai-agentml-agentml)
  - [2.3 &lt;assign&gt;](#element-github-com-agentflare-ai-agentml-assign)
  - [2.4 &lt;cancel&gt;](#element-github-com-agentflare-ai-agentml-cancel)
  - [2.5 &lt;content&gt;](#element-github-com-agentflare-ai-agentml-content)
  - [2.6 &lt;data&gt;](#element-github-com-agentflare-ai-agentml-data)
  - [2.7 &lt;datamodel&gt;](#element-github-com-agentflare-ai-agentml-datamodel)
  - [2.8 &lt;donedata&gt;](#element-github-com-agentflare-ai-agentml-donedata)
  - [2.9 &lt;else&gt;](#element-github-com-agentflare-ai-agentml-else)
  - [2.10 &lt;elseif&gt;](#element-github-com-agentflare-ai-agentml-elseif)
  - [2.11 &lt;executable&gt;](#element-github-com-agentflare-ai-agentml-executable)
  - [2.12 &lt;final&gt;](#element-github-com-agentflare-ai-agentml-final)
  - [2.13 &lt;finalize&gt;](#element-github-com-agentflare-ai-agentml-finalize)
  - [2.14 &lt;foreach&gt;](#element-github-com-agentflare-ai-agentml-foreach)
  - [2.15 &lt;history&gt;](#element-github-com-agentflare-ai-agentml-history)
  - [2.16 &lt;if&gt;](#element-github-com-agentflare-ai-agentml-if)
  - [2.17 &lt;initial&gt;](#element-github-com-agentflare-ai-agentml-initial)
  - [2.18 &lt;invokable&gt;](#element-github-com-agentflare-ai-agentml-invokable)
  - [2.19 &lt;invoke&gt;](#element-github-com-agentflare-ai-agentml-invoke)
  - [2.20 &lt;log&gt;](#element-github-com-agentflare-ai-agentml-log)
  - [2.21 &lt;onentry&gt;](#element-github-com-agentflare-ai-agentml-onentry)
  - [2.22 &lt;onexit&gt;](#element-github-com-agentflare-ai-agentml-onexit)
  - [2.23 &lt;parallel&gt;](#element-github-com-agentflare-ai-agentml-parallel)
  - [2.24 &lt;param&gt;](#element-github-com-agentflare-ai-agentml-param)
  - [2.25 &lt;raise&gt;](#element-github-com-agentflare-ai-agentml-raise)
  - [2.26 &lt;root&gt;](#element-github-com-agentflare-ai-agentml-root)
  - [2.27 &lt;script&gt;](#element-github-com-agentflare-ai-agentml-script)
  - [2.28 &lt;send&gt;](#element-github-com-agentflare-ai-agentml-send)
  - [2.29 &lt;state&gt;](#element-github-com-agentflare-ai-agentml-state)
  - [2.30 &lt;transition&gt;](#element-github-com-agentflare-ai-agentml-transition)
- [3 Types](#types)
  - [3.1 AssignType.datatype](#type-github-com-agentflare-ai-agentml-assigntype-datatype)
  - [3.2 Binding.datatype](#type-github-com-agentflare-ai-agentml-binding-datatype)
  - [3.3 Boolean.datatype](#type-github-com-agentflare-ai-agentml-boolean-datatype)
  - [3.4 CondLang.datatype](#type-github-com-agentflare-ai-agentml-condlang-datatype)
  - [3.5 Duration.datatype](#type-github-com-agentflare-ai-agentml-duration-datatype)
  - [3.6 Encode.datatype](#type-github-com-agentflare-ai-agentml-encode-datatype)
  - [3.7 EventType.datatype](#type-github-com-agentflare-ai-agentml-eventtype-datatype)
  - [3.8 EventTypes.datatype](#type-github-com-agentflare-ai-agentml-eventtypes-datatype)
  - [3.9 Exmode.datatype](#type-github-com-agentflare-ai-agentml-exmode-datatype)
  - [3.10 HistoryType.datatype](#type-github-com-agentflare-ai-agentml-historytype-datatype)
  - [3.11 Integer.datatype](#type-github-com-agentflare-ai-agentml-integer-datatype)
  - [3.12 JsonSchema.datatype](#type-github-com-agentflare-ai-agentml-jsonschema-datatype)
  - [3.13 LocLang.datatype](#type-github-com-agentflare-ai-agentml-loclang-datatype)
  - [3.14 TransitionType.datatype](#type-github-com-agentflare-ai-agentml-transitiontype-datatype)
  - [3.15 URI.datatype](#type-github-com-agentflare-ai-agentml-uri-datatype)
  - [3.16 ValueLang.datatype](#type-github-com-agentflare-ai-agentml-valuelang-datatype)
  - [3.17 agentml.agentml.type](#type-github-com-agentflare-ai-agentml-agentml-agentml-type)
  - [3.18 agentml.assign.type](#type-github-com-agentflare-ai-agentml-agentml-assign-type)
  - [3.19 agentml.cancel.type](#type-github-com-agentflare-ai-agentml-agentml-cancel-type)
  - [3.20 agentml.content.type](#type-github-com-agentflare-ai-agentml-agentml-content-type)
  - [3.21 agentml.data.type](#type-github-com-agentflare-ai-agentml-agentml-data-type)
  - [3.22 agentml.datamodel.type](#type-github-com-agentflare-ai-agentml-agentml-datamodel-type)
  - [3.23 agentml.donedata.type](#type-github-com-agentflare-ai-agentml-agentml-donedata-type)
  - [3.24 agentml.else.type](#type-github-com-agentflare-ai-agentml-agentml-else-type)
  - [3.25 agentml.elseif.type](#type-github-com-agentflare-ai-agentml-agentml-elseif-type)
  - [3.26 agentml.final.type](#type-github-com-agentflare-ai-agentml-agentml-final-type)
  - [3.27 agentml.finalize.type](#type-github-com-agentflare-ai-agentml-agentml-finalize-type)
  - [3.28 agentml.foreach.type](#type-github-com-agentflare-ai-agentml-agentml-foreach-type)
  - [3.29 agentml.history.type](#type-github-com-agentflare-ai-agentml-agentml-history-type)
  - [3.30 agentml.if.type](#type-github-com-agentflare-ai-agentml-agentml-if-type)
  - [3.31 agentml.initial.type](#type-github-com-agentflare-ai-agentml-agentml-initial-type)
  - [3.32 agentml.invoke.type](#type-github-com-agentflare-ai-agentml-agentml-invoke-type)
  - [3.33 agentml.log.type](#type-github-com-agentflare-ai-agentml-agentml-log-type)
  - [3.34 agentml.onentry.type](#type-github-com-agentflare-ai-agentml-agentml-onentry-type)
  - [3.35 agentml.onexit.type](#type-github-com-agentflare-ai-agentml-agentml-onexit-type)
  - [3.36 agentml.parallel.type](#type-github-com-agentflare-ai-agentml-agentml-parallel-type)
  - [3.37 agentml.param.type](#type-github-com-agentflare-ai-agentml-agentml-param-type)
  - [3.38 agentml.raise.type](#type-github-com-agentflare-ai-agentml-agentml-raise-type)
  - [3.39 agentml.script.type](#type-github-com-agentflare-ai-agentml-agentml-script-type)
  - [3.40 agentml.send.type](#type-github-com-agentflare-ai-agentml-agentml-send-type)
  - [3.41 agentml.state.type](#type-github-com-agentflare-ai-agentml-agentml-state-type)
  - [3.42 agentml.transition.type](#type-github-com-agentflare-ai-agentml-agentml-transition-type)
- [ Identity Constraints](#identity-constraints)

## 1 Overview

- **Target namespace:** `github.com/agentflare-ai/agentml`
- **Elements:** 30
- **Types:** 42

## 2 Elements

<a id="element-github-com-agentflare-ai-agentml-agent"></a>
### 2.1 &lt;agent&gt;

**Description**
```
Backwards-compatible alias for the `agentml` root element; it shares
                the same attributes and content model and is interpreted identically.
```

- **Namespace:** github.com/agentflare-ai/agentml
- **Type:** [agentml.agentml.type](#type-github-com-agentflare-ai-agentml-agentml-agentml-type)
- **Cardinality:** exactly 1
- **Extensibility:** allows any elements (namespace: ##other)

| Attribute | Use | Type | Default | Fixed |
|-----------|-----|------|---------|-------|
| `binding` | optional | [Binding.datatype](#type-github-com-agentflare-ai-agentml-binding-datatype) | — | — |
| `datamodel` | optional | `xs:NMTOKEN` | null | — |
| `initial` | optional | `xs:IDREFS` | — | — |
| `name` | optional | `xs:NMTOKEN` | — | — |
| `version` | required | `xs:decimal` | — | 1.0 |

#### 2.1.2 Children

&lt;state&gt; Basic AgentML state. May be atomic (no child states) or compound (containing nested `state`, `parallel`, and/or `final` elements). Supports entry/exit actions, event-triggered transitions, history pseudo-states, state-local datamodel, and invocations of external services. Occurs 0 or more times. See [2.29 &lt;state&gt;](#element-github-com-agentflare-ai-agentml-state).
&lt;parallel&gt; Child element. Occurs 0 or more times. See [2.23 &lt;parallel&gt;](#element-github-com-agentflare-ai-agentml-parallel).
&lt;final&gt; Child element. Occurs 0 or more times. See [2.12 &lt;final&gt;](#element-github-com-agentflare-ai-agentml-final).
&lt;datamodel&gt; Child element. Occurs 0 or more times. See [2.7 &lt;datamodel&gt;](#element-github-com-agentflare-ai-agentml-datamodel).
&lt;script&gt; Child element. Occurs 0 or more times. See [2.27 &lt;script&gt;](#element-github-com-agentflare-ai-agentml-script).

<a id="element-github-com-agentflare-ai-agentml-agentml"></a>
### 2.2 &lt;agentml&gt;

**Description**
```
Root AgentML state machine document. Declares version, binding,
                datamodel implementation, and root-level states, parallel regions, final states,
                datamodel definitions, and scripts. Also supports JSON Schema declarations via
                `schema:*` attributes for validating datamodel variables and event payloads.
```

- **Namespace:** github.com/agentflare-ai/agentml
- **Type:** [agentml.agentml.type](#type-github-com-agentflare-ai-agentml-agentml-agentml-type)
- **Cardinality:** exactly 1
- **Extensibility:** allows any elements (namespace: ##other)

| Attribute | Use | Type | Default | Fixed |
|-----------|-----|------|---------|-------|
| `binding` | optional | [Binding.datatype](#type-github-com-agentflare-ai-agentml-binding-datatype) | — | — |
| `datamodel` | optional | `xs:NMTOKEN` | null | — |
| `initial` | optional | `xs:IDREFS` | — | — |
| `name` | optional | `xs:NMTOKEN` | — | — |
| `version` | required | `xs:decimal` | — | 1.0 |

#### 2.2.2 Children

&lt;state&gt; Basic AgentML state. May be atomic (no child states) or compound (containing nested `state`, `parallel`, and/or `final` elements). Supports entry/exit actions, event-triggered transitions, history pseudo-states, state-local datamodel, and invocations of external services. Occurs 0 or more times. See [2.29 &lt;state&gt;](#element-github-com-agentflare-ai-agentml-state).
&lt;parallel&gt; Child element. Occurs 0 or more times. See [2.23 &lt;parallel&gt;](#element-github-com-agentflare-ai-agentml-parallel).
&lt;final&gt; Child element. Occurs 0 or more times. See [2.12 &lt;final&gt;](#element-github-com-agentflare-ai-agentml-final).
&lt;datamodel&gt; Child element. Occurs 0 or more times. See [2.7 &lt;datamodel&gt;](#element-github-com-agentflare-ai-agentml-datamodel).
&lt;script&gt; Child element. Occurs 0 or more times. See [2.27 &lt;script&gt;](#element-github-com-agentflare-ai-agentml-script).

<a id="element-github-com-agentflare-ai-agentml-assign"></a>
### 2.3 &lt;assign&gt;

- **Namespace:** github.com/agentflare-ai/agentml
- **Type:** [agentml.assign.type](#type-github-com-agentflare-ai-agentml-agentml-assign-type)
- **Cardinality:** exactly 1
- **Extensibility:** allows any elements (namespace: ##any)

| Attribute | Use | Type | Default | Fixed |
|-----------|-----|------|---------|-------|
| `attr` | optional | `xs:NMTOKEN` | — | — |
| `expr` | optional | [ValueLang.datatype](#type-github-com-agentflare-ai-agentml-valuelang-datatype) | — | — |
| `location` | required | [LocLang.datatype](#type-github-com-agentflare-ai-agentml-loclang-datatype) | — | — |
| `type` | optional | [AssignType.datatype](#type-github-com-agentflare-ai-agentml-assigntype-datatype) | replacechildren | — |

<a id="element-github-com-agentflare-ai-agentml-cancel"></a>
### 2.4 &lt;cancel&gt;

- **Namespace:** github.com/agentflare-ai/agentml
- **Type:** [agentml.cancel.type](#type-github-com-agentflare-ai-agentml-agentml-cancel-type)
- **Cardinality:** exactly 1
- **Extensibility:** allows any elements (namespace: ##other)

| Attribute | Use | Type | Default | Fixed |
|-----------|-----|------|---------|-------|
| `sendid` | optional | `xs:IDREF` | — | — |
| `sendidexpr` | optional | [ValueLang.datatype](#type-github-com-agentflare-ai-agentml-valuelang-datatype) | — | — |

<a id="element-github-com-agentflare-ai-agentml-content"></a>
### 2.5 &lt;content&gt;

- **Namespace:** github.com/agentflare-ai/agentml
- **Type:** [agentml.content.type](#type-github-com-agentflare-ai-agentml-agentml-content-type)
- **Cardinality:** exactly 1
- **Extensibility:** allows any elements (namespace: ##any)

| Attribute | Use | Type | Default | Fixed |
|-----------|-----|------|---------|-------|
| `expr` | optional | [ValueLang.datatype](#type-github-com-agentflare-ai-agentml-valuelang-datatype) | — | — |

<a id="element-github-com-agentflare-ai-agentml-data"></a>
### 2.6 &lt;data&gt;

- **Namespace:** github.com/agentflare-ai/agentml
- **Type:** [agentml.data.type](#type-github-com-agentflare-ai-agentml-agentml-data-type)
- **Cardinality:** exactly 1
- **Extensibility:** allows any elements (namespace: ##any)

| Attribute | Use | Type | Default | Fixed |
|-----------|-----|------|---------|-------|
| `encoding` | optional | [Encode.datatype](#type-github-com-agentflare-ai-agentml-encode-datatype) | json | — |
| `expr` | optional | [ValueLang.datatype](#type-github-com-agentflare-ai-agentml-valuelang-datatype) | — | — |
| `id` | required | `xs:ID` | — | — |
| `schema` | optional | [JsonSchema.datatype](#type-github-com-agentflare-ai-agentml-jsonschema-datatype) | — | — |
| `src` | optional | [URI.datatype](#type-github-com-agentflare-ai-agentml-uri-datatype) | — | — |

<a id="element-github-com-agentflare-ai-agentml-datamodel"></a>
### 2.7 &lt;datamodel&gt;

- **Namespace:** github.com/agentflare-ai/agentml
- **Type:** [agentml.datamodel.type](#type-github-com-agentflare-ai-agentml-agentml-datamodel-type)
- **Cardinality:** exactly 1
- **Extensibility:** allows any elements (namespace: ##other)


#### 2.7.2 Children

&lt;data&gt; Child element. Occurs 0 or more times. See [2.6 &lt;data&gt;](#element-github-com-agentflare-ai-agentml-data).

<a id="element-github-com-agentflare-ai-agentml-donedata"></a>
### 2.8 &lt;donedata&gt;

- **Namespace:** github.com/agentflare-ai/agentml
- **Type:** [agentml.donedata.type](#type-github-com-agentflare-ai-agentml-agentml-donedata-type)
- **Cardinality:** exactly 1


#### 2.8.2 Children

&lt;content&gt; Child element. Occurs 0 or 1 times. See [2.5 &lt;content&gt;](#element-github-com-agentflare-ai-agentml-content).
&lt;param&gt; Child element. Occurs 0 or more times. See [2.24 &lt;param&gt;](#element-github-com-agentflare-ai-agentml-param).

<a id="element-github-com-agentflare-ai-agentml-else"></a>
### 2.9 &lt;else&gt;

- **Namespace:** github.com/agentflare-ai/agentml
- **Type:** [agentml.else.type](#type-github-com-agentflare-ai-agentml-agentml-else-type)
- **Cardinality:** exactly 1


<a id="element-github-com-agentflare-ai-agentml-elseif"></a>
### 2.10 &lt;elseif&gt;

- **Namespace:** github.com/agentflare-ai/agentml
- **Type:** [agentml.elseif.type](#type-github-com-agentflare-ai-agentml-agentml-elseif-type)
- **Cardinality:** exactly 1

| Attribute | Use | Type | Default | Fixed |
|-----------|-----|------|---------|-------|
| `cond` | required | [CondLang.datatype](#type-github-com-agentflare-ai-agentml-condlang-datatype) | — | — |

<a id="element-github-com-agentflare-ai-agentml-executable"></a>
### 2.11 &lt;executable&gt;

**Description**
```
Abstract base for executable content. Extend with
                substitutionGroup="agentml:executable"
```

- **Namespace:** github.com/agentflare-ai/agentml
- **Cardinality:** exactly 1


<a id="element-github-com-agentflare-ai-agentml-final"></a>
### 2.12 &lt;final&gt;

- **Namespace:** github.com/agentflare-ai/agentml
- **Type:** [agentml.final.type](#type-github-com-agentflare-ai-agentml-agentml-final-type)
- **Cardinality:** exactly 1
- **Extensibility:** allows any elements (namespace: ##other)

| Attribute | Use | Type | Default | Fixed |
|-----------|-----|------|---------|-------|
| `id` | optional | `xs:ID` | — | — |

#### 2.12.2 Children

&lt;onentry&gt; Executable content that runs whenever a state is entered, after entry transitions have been taken. Multiple `onentry` elements are executed in document order. Occurs 0 or more times. See [2.21 &lt;onentry&gt;](#element-github-com-agentflare-ai-agentml-onentry).
&lt;onexit&gt; Executable content that runs whenever a state is exited, before any exit transitions are taken. Multiple `onexit` elements are executed in document order. Occurs 0 or more times. See [2.22 &lt;onexit&gt;](#element-github-com-agentflare-ai-agentml-onexit).
&lt;donedata&gt; Child element. Occurs 0 or 1 times. See [2.8 &lt;donedata&gt;](#element-github-com-agentflare-ai-agentml-donedata).

<a id="element-github-com-agentflare-ai-agentml-finalize"></a>
### 2.13 &lt;finalize&gt;

- **Namespace:** github.com/agentflare-ai/agentml
- **Type:** [agentml.finalize.type](#type-github-com-agentflare-ai-agentml-agentml-finalize-type)
- **Cardinality:** exactly 1


#### 2.13.2 Children

&lt;executable&gt; Abstract base for executable content. Extend with substitutionGroup="agentml:executable" Occurs 0 or more times. See [2.11 &lt;executable&gt;](#element-github-com-agentflare-ai-agentml-executable).

<a id="element-github-com-agentflare-ai-agentml-foreach"></a>
### 2.14 &lt;foreach&gt;

- **Namespace:** github.com/agentflare-ai/agentml
- **Type:** [agentml.foreach.type](#type-github-com-agentflare-ai-agentml-agentml-foreach-type)
- **Cardinality:** exactly 1

| Attribute | Use | Type | Default | Fixed |
|-----------|-----|------|---------|-------|
| `array` | required | [ValueLang.datatype](#type-github-com-agentflare-ai-agentml-valuelang-datatype) | — | — |
| `index` | optional | `xs:string` | — | — |
| `item` | required | `xs:string` | — | — |

#### 2.14.2 Children

&lt;executable&gt; Abstract base for executable content. Extend with substitutionGroup="agentml:executable" Occurs 0 or more times. See [2.11 &lt;executable&gt;](#element-github-com-agentflare-ai-agentml-executable).

<a id="element-github-com-agentflare-ai-agentml-history"></a>
### 2.15 &lt;history&gt;

- **Namespace:** github.com/agentflare-ai/agentml
- **Type:** [agentml.history.type](#type-github-com-agentflare-ai-agentml-agentml-history-type)
- **Cardinality:** exactly 1
- **Extensibility:** allows any elements (namespace: ##other)

| Attribute | Use | Type | Default | Fixed |
|-----------|-----|------|---------|-------|
| `id` | optional | `xs:ID` | — | — |
| `type` | optional | [HistoryType.datatype](#type-github-com-agentflare-ai-agentml-historytype-datatype) | — | — |

#### 2.15.2 Children

&lt;transition&gt; Event-triggered state change. A transition is selected when its `event` pattern matches the current event and its optional `cond` expression evaluates to true. The `target` attribute selects the destination state configuration, and `type` controls whether the transition is internal or external. The optional `schema` attribute can reference a JSON Schema used to validate the associated event payload. Occurs 1 time. See [2.30 &lt;transition&gt;](#element-github-com-agentflare-ai-agentml-transition).

<a id="element-github-com-agentflare-ai-agentml-if"></a>
### 2.16 &lt;if&gt;

- **Namespace:** github.com/agentflare-ai/agentml
- **Type:** [agentml.if.type](#type-github-com-agentflare-ai-agentml-agentml-if-type)
- **Cardinality:** exactly 1

| Attribute | Use | Type | Default | Fixed |
|-----------|-----|------|---------|-------|
| `cond` | required | [CondLang.datatype](#type-github-com-agentflare-ai-agentml-condlang-datatype) | — | — |

#### 2.16.2 Children

&lt;executable&gt; Abstract base for executable content. Extend with substitutionGroup="agentml:executable" Occurs 0 or more times. See [2.11 &lt;executable&gt;](#element-github-com-agentflare-ai-agentml-executable).
&lt;elseif&gt; Child element. Occurs 1 time. See [2.10 &lt;elseif&gt;](#element-github-com-agentflare-ai-agentml-elseif).
&lt;else&gt; Child element. Occurs 1 time. See [2.9 &lt;else&gt;](#element-github-com-agentflare-ai-agentml-else).

<a id="element-github-com-agentflare-ai-agentml-initial"></a>
### 2.17 &lt;initial&gt;

**Description**
```
Default transition for a compound state. When the parent state is
                entered without an explicit target, the `transition` child of this `initial` element
                is used to determine the starting child configuration.
```

- **Namespace:** github.com/agentflare-ai/agentml
- **Type:** [agentml.initial.type](#type-github-com-agentflare-ai-agentml-agentml-initial-type)
- **Cardinality:** exactly 1
- **Extensibility:** allows any elements (namespace: ##other)


#### 2.17.2 Children

&lt;transition&gt; Event-triggered state change. A transition is selected when its `event` pattern matches the current event and its optional `cond` expression evaluates to true. The `target` attribute selects the destination state configuration, and `type` controls whether the transition is internal or external. The optional `schema` attribute can reference a JSON Schema used to validate the associated event payload. Occurs 1 time. See [2.30 &lt;transition&gt;](#element-github-com-agentflare-ai-agentml-transition).

<a id="element-github-com-agentflare-ai-agentml-invokable"></a>
### 2.18 &lt;invokable&gt;

**Description**
```
Abstract base for invokable elements. Extend with
                substitutionGroup="agentml:invokable"
```

- **Namespace:** github.com/agentflare-ai/agentml
- **Cardinality:** exactly 1


<a id="element-github-com-agentflare-ai-agentml-invoke"></a>
### 2.19 &lt;invoke&gt;

- **Namespace:** github.com/agentflare-ai/agentml
- **Type:** [agentml.invoke.type](#type-github-com-agentflare-ai-agentml-agentml-invoke-type)
- **Cardinality:** exactly 1
- **Extensibility:** allows any elements (namespace: ##other)

| Attribute | Use | Type | Default | Fixed |
|-----------|-----|------|---------|-------|
| `autoforward` | optional | [Boolean.datatype](#type-github-com-agentflare-ai-agentml-boolean-datatype) | false | — |
| `id` | optional | `xs:ID` | — | — |
| `idlocation` | optional | [LocLang.datatype](#type-github-com-agentflare-ai-agentml-loclang-datatype) | — | — |
| `namelist` | optional | `xs:string` | — | — |
| `src` | optional | [URI.datatype](#type-github-com-agentflare-ai-agentml-uri-datatype) | — | — |
| `srcexpr` | optional | [ValueLang.datatype](#type-github-com-agentflare-ai-agentml-valuelang-datatype) | — | — |
| `type` | optional | `xs:string` | agentml | — |
| `typeexpr` | optional | [ValueLang.datatype](#type-github-com-agentflare-ai-agentml-valuelang-datatype) | — | — |

#### 2.19.2 Children

&lt;content&gt; Child element. Occurs 0 or 1 times. See [2.5 &lt;content&gt;](#element-github-com-agentflare-ai-agentml-content).
&lt;param&gt; Child element. Occurs 0 or more times. See [2.24 &lt;param&gt;](#element-github-com-agentflare-ai-agentml-param).
&lt;finalize&gt; Child element. Occurs 0 or 1 times. See [2.13 &lt;finalize&gt;](#element-github-com-agentflare-ai-agentml-finalize).

<a id="element-github-com-agentflare-ai-agentml-log"></a>
### 2.20 &lt;log&gt;

- **Namespace:** github.com/agentflare-ai/agentml
- **Type:** [agentml.log.type](#type-github-com-agentflare-ai-agentml-agentml-log-type)
- **Cardinality:** exactly 1
- **Extensibility:** allows any elements (namespace: ##other)

| Attribute | Use | Type | Default | Fixed |
|-----------|-----|------|---------|-------|
| `expr` | optional | [ValueLang.datatype](#type-github-com-agentflare-ai-agentml-valuelang-datatype) | — | — |
| `label` | optional | `xs:string` | — | — |

<a id="element-github-com-agentflare-ai-agentml-onentry"></a>
### 2.21 &lt;onentry&gt;

**Description**
```
Executable content that runs whenever a state is entered, after
                entry transitions have been taken. Multiple `onentry` elements are executed in
                document order.
```

- **Namespace:** github.com/agentflare-ai/agentml
- **Type:** [agentml.onentry.type](#type-github-com-agentflare-ai-agentml-agentml-onentry-type)
- **Cardinality:** exactly 1


#### 2.21.2 Children

&lt;executable&gt; Abstract base for executable content. Extend with substitutionGroup="agentml:executable" Occurs 0 or more times. See [2.11 &lt;executable&gt;](#element-github-com-agentflare-ai-agentml-executable).

<a id="element-github-com-agentflare-ai-agentml-onexit"></a>
### 2.22 &lt;onexit&gt;

**Description**
```
Executable content that runs whenever a state is exited, before any
                exit transitions are taken. Multiple `onexit` elements are executed in document
                order.
```

- **Namespace:** github.com/agentflare-ai/agentml
- **Type:** [agentml.onexit.type](#type-github-com-agentflare-ai-agentml-agentml-onexit-type)
- **Cardinality:** exactly 1


#### 2.22.2 Children

&lt;executable&gt; Abstract base for executable content. Extend with substitutionGroup="agentml:executable" Occurs 0 or more times. See [2.11 &lt;executable&gt;](#element-github-com-agentflare-ai-agentml-executable).

<a id="element-github-com-agentflare-ai-agentml-parallel"></a>
### 2.23 &lt;parallel&gt;

- **Namespace:** github.com/agentflare-ai/agentml
- **Type:** [agentml.parallel.type](#type-github-com-agentflare-ai-agentml-agentml-parallel-type)
- **Cardinality:** exactly 1

| Attribute | Use | Type | Default | Fixed |
|-----------|-----|------|---------|-------|
| `id` | optional | `xs:ID` | — | — |

#### 2.23.2 Children

&lt;onentry&gt; Executable content that runs whenever a state is entered, after entry transitions have been taken. Multiple `onentry` elements are executed in document order. Occurs 0 or more times. See [2.21 &lt;onentry&gt;](#element-github-com-agentflare-ai-agentml-onentry).
&lt;onexit&gt; Executable content that runs whenever a state is exited, before any exit transitions are taken. Multiple `onexit` elements are executed in document order. Occurs 0 or more times. See [2.22 &lt;onexit&gt;](#element-github-com-agentflare-ai-agentml-onexit).
&lt;transition&gt; Event-triggered state change. A transition is selected when its `event` pattern matches the current event and its optional `cond` expression evaluates to true. The `target` attribute selects the destination state configuration, and `type` controls whether the transition is internal or external. The optional `schema` attribute can reference a JSON Schema used to validate the associated event payload. Occurs 0 or more times. See [2.30 &lt;transition&gt;](#element-github-com-agentflare-ai-agentml-transition).
&lt;state&gt; Basic AgentML state. May be atomic (no child states) or compound (containing nested `state`, `parallel`, and/or `final` elements). Supports entry/exit actions, event-triggered transitions, history pseudo-states, state-local datamodel, and invocations of external services. Occurs 0 or more times. See [2.29 &lt;state&gt;](#element-github-com-agentflare-ai-agentml-state).
&lt;parallel&gt; Child element. Occurs 0 or more times. See [2.23 &lt;parallel&gt;](#element-github-com-agentflare-ai-agentml-parallel).
&lt;history&gt; Child element. Occurs 0 or more times. See [2.15 &lt;history&gt;](#element-github-com-agentflare-ai-agentml-history).
&lt;datamodel&gt; Child element. Occurs 0 or 1 times. See [2.7 &lt;datamodel&gt;](#element-github-com-agentflare-ai-agentml-datamodel).
&lt;invokable&gt; Abstract base for invokable elements. Extend with substitutionGroup="agentml:invokable" Occurs 0 or more times. See [2.18 &lt;invokable&gt;](#element-github-com-agentflare-ai-agentml-invokable).

<a id="element-github-com-agentflare-ai-agentml-param"></a>
### 2.24 &lt;param&gt;

- **Namespace:** github.com/agentflare-ai/agentml
- **Type:** [agentml.param.type](#type-github-com-agentflare-ai-agentml-agentml-param-type)
- **Cardinality:** exactly 1
- **Extensibility:** allows any elements (namespace: ##other)

| Attribute | Use | Type | Default | Fixed |
|-----------|-----|------|---------|-------|
| `expr` | optional | [ValueLang.datatype](#type-github-com-agentflare-ai-agentml-valuelang-datatype) | — | — |
| `location` | optional | [LocLang.datatype](#type-github-com-agentflare-ai-agentml-loclang-datatype) | — | — |
| `name` | required | `xs:NMTOKEN` | — | — |

<a id="element-github-com-agentflare-ai-agentml-raise"></a>
### 2.25 &lt;raise&gt;

- **Namespace:** github.com/agentflare-ai/agentml
- **Type:** [agentml.raise.type](#type-github-com-agentflare-ai-agentml-agentml-raise-type)
- **Cardinality:** exactly 1

| Attribute | Use | Type | Default | Fixed |
|-----------|-----|------|---------|-------|
| `event` | required | `xs:NMTOKEN` | — | — |

<a id="element-github-com-agentflare-ai-agentml-root"></a>
### 2.26 &lt;root&gt;

**Description**
```
Abstract base for root elements. Extend with
                substitutionGroup="agentml:root"
```

- **Namespace:** github.com/agentflare-ai/agentml
- **Cardinality:** exactly 1


<a id="element-github-com-agentflare-ai-agentml-script"></a>
### 2.27 &lt;script&gt;

- **Namespace:** github.com/agentflare-ai/agentml
- **Type:** [agentml.script.type](#type-github-com-agentflare-ai-agentml-agentml-script-type)
- **Cardinality:** exactly 1
- **Extensibility:** allows any elements (namespace: ##other)

| Attribute | Use | Type | Default | Fixed |
|-----------|-----|------|---------|-------|
| `src` | optional | [URI.datatype](#type-github-com-agentflare-ai-agentml-uri-datatype) | — | — |

<a id="element-github-com-agentflare-ai-agentml-send"></a>
### 2.28 &lt;send&gt;

- **Namespace:** github.com/agentflare-ai/agentml
- **Type:** [agentml.send.type](#type-github-com-agentflare-ai-agentml-agentml-send-type)
- **Cardinality:** exactly 1
- **Extensibility:** allows any elements (namespace: ##other)

| Attribute | Use | Type | Default | Fixed |
|-----------|-----|------|---------|-------|
| `delay` | optional | [Duration.datatype](#type-github-com-agentflare-ai-agentml-duration-datatype) | 0s | — |
| `delayexpr` | optional | [ValueLang.datatype](#type-github-com-agentflare-ai-agentml-valuelang-datatype) | — | — |
| `event` | optional | [EventType.datatype](#type-github-com-agentflare-ai-agentml-eventtype-datatype) | — | — |
| `eventexpr` | optional | [ValueLang.datatype](#type-github-com-agentflare-ai-agentml-valuelang-datatype) | — | — |
| `id` | optional | `xs:ID` | — | — |
| `idlocation` | optional | [LocLang.datatype](#type-github-com-agentflare-ai-agentml-loclang-datatype) | — | — |
| `namelist` | optional | `xs:string` | — | — |
| `target` | optional | [URI.datatype](#type-github-com-agentflare-ai-agentml-uri-datatype) | — | — |
| `targetexpr` | optional | [ValueLang.datatype](#type-github-com-agentflare-ai-agentml-valuelang-datatype) | — | — |
| `type` | optional | `xs:string` | agentml | — |
| `typeexpr` | optional | [ValueLang.datatype](#type-github-com-agentflare-ai-agentml-valuelang-datatype) | — | — |

#### 2.28.2 Children

&lt;content&gt; Child element. Occurs 0 or 1 times. See [2.5 &lt;content&gt;](#element-github-com-agentflare-ai-agentml-content).
&lt;param&gt; Child element. Occurs 0 or more times. See [2.24 &lt;param&gt;](#element-github-com-agentflare-ai-agentml-param).

<a id="element-github-com-agentflare-ai-agentml-state"></a>
### 2.29 &lt;state&gt;

**Description**
```
Basic AgentML state. May be atomic (no child states) or compound
                (containing nested `state`, `parallel`, and/or `final` elements). Supports
                entry/exit actions, event-triggered transitions, history pseudo-states, state-local
                datamodel, and invocations of external services.
```

- **Namespace:** github.com/agentflare-ai/agentml
- **Type:** [agentml.state.type](#type-github-com-agentflare-ai-agentml-agentml-state-type)
- **Cardinality:** exactly 1

| Attribute | Use | Type | Default | Fixed |
|-----------|-----|------|---------|-------|
| `id` | optional | `xs:ID` | — | — |
| `initial` | optional | `xs:IDREFS` | — | — |

#### 2.29.2 Children

&lt;onentry&gt; Executable content that runs whenever a state is entered, after entry transitions have been taken. Multiple `onentry` elements are executed in document order. Occurs 0 or more times. See [2.21 &lt;onentry&gt;](#element-github-com-agentflare-ai-agentml-onentry).
&lt;onexit&gt; Executable content that runs whenever a state is exited, before any exit transitions are taken. Multiple `onexit` elements are executed in document order. Occurs 0 or more times. See [2.22 &lt;onexit&gt;](#element-github-com-agentflare-ai-agentml-onexit).
&lt;transition&gt; Event-triggered state change. A transition is selected when its `event` pattern matches the current event and its optional `cond` expression evaluates to true. The `target` attribute selects the destination state configuration, and `type` controls whether the transition is internal or external. The optional `schema` attribute can reference a JSON Schema used to validate the associated event payload. Occurs 0 or more times. See [2.30 &lt;transition&gt;](#element-github-com-agentflare-ai-agentml-transition).
&lt;initial&gt; Default transition for a compound state. When the parent state is entered without an explicit target, the `transition` child of this `initial` element is used to determine the starting child configuration. Occurs 0 or 1 times. See [2.17 &lt;initial&gt;](#element-github-com-agentflare-ai-agentml-initial).
&lt;state&gt; Basic AgentML state. May be atomic (no child states) or compound (containing nested `state`, `parallel`, and/or `final` elements). Supports entry/exit actions, event-triggered transitions, history pseudo-states, state-local datamodel, and invocations of external services. Occurs 0 or more times. See [2.29 &lt;state&gt;](#element-github-com-agentflare-ai-agentml-state).
&lt;parallel&gt; Child element. Occurs 0 or more times. See [2.23 &lt;parallel&gt;](#element-github-com-agentflare-ai-agentml-parallel).
&lt;final&gt; Child element. Occurs 0 or more times. See [2.12 &lt;final&gt;](#element-github-com-agentflare-ai-agentml-final).
&lt;history&gt; Child element. Occurs 0 or more times. See [2.15 &lt;history&gt;](#element-github-com-agentflare-ai-agentml-history).
&lt;datamodel&gt; Child element. Occurs 0 or 1 times. See [2.7 &lt;datamodel&gt;](#element-github-com-agentflare-ai-agentml-datamodel).
&lt;invokable&gt; Abstract base for invokable elements. Extend with substitutionGroup="agentml:invokable" Occurs 0 or more times. See [2.18 &lt;invokable&gt;](#element-github-com-agentflare-ai-agentml-invokable).

<a id="element-github-com-agentflare-ai-agentml-transition"></a>
### 2.30 &lt;transition&gt;

**Description**
```
Event-triggered state change. A transition is selected when its
                `event` pattern matches the current event and its optional `cond` expression
                evaluates to true. The `target` attribute selects the destination state
                configuration, and `type` controls whether the transition is internal or external.
                The optional `schema` attribute can reference a JSON Schema used to validate the
                associated event payload.
```

- **Namespace:** github.com/agentflare-ai/agentml
- **Type:** [agentml.transition.type](#type-github-com-agentflare-ai-agentml-agentml-transition-type)
- **Cardinality:** exactly 1

| Attribute | Use | Type | Default | Fixed |
|-----------|-----|------|---------|-------|
| `cond` | optional | [CondLang.datatype](#type-github-com-agentflare-ai-agentml-condlang-datatype) | — | — |
| `event` | optional | [EventTypes.datatype](#type-github-com-agentflare-ai-agentml-eventtypes-datatype) | — | — |
| `schema` | optional | [JsonSchema.datatype](#type-github-com-agentflare-ai-agentml-jsonschema-datatype) | — | — |
| `target` | optional | `xs:IDREFS` | — | — |
| `type` | optional | [TransitionType.datatype](#type-github-com-agentflare-ai-agentml-transitiontype-datatype) | — | — |

#### 2.30.2 Children

&lt;executable&gt; Abstract base for executable content. Extend with substitutionGroup="agentml:executable" Occurs 0 or more times. See [2.11 &lt;executable&gt;](#element-github-com-agentflare-ai-agentml-executable).

## 3 Types

<a id="type-github-com-agentflare-ai-agentml-assigntype-datatype"></a>
### 3.1 AssignType.datatype

**Description**
```
Assignment operation type for datamodel manipulation
```

- **Kind:** simple
- **Namespace:** github.com/agentflare-ai/agentml

| Facet | Value |
|-------|-------|
| enumeration | replacechildren, firstchild, lastchild, previoussibling, nextsibling, replace, delete, addattribute |

<a id="type-github-com-agentflare-ai-agentml-binding-datatype"></a>
### 3.2 Binding.datatype

**Description**
```
Datamodel binding: "early" (load-time init) or "late" (lazy init)
```

- **Kind:** simple
- **Namespace:** github.com/agentflare-ai/agentml

| Facet | Value |
|-------|-------|
| enumeration | early, late |

<a id="type-github-com-agentflare-ai-agentml-boolean-datatype"></a>
### 3.3 Boolean.datatype

**Description**
```
Boolean values: "true" or "false"
```

- **Kind:** simple
- **Namespace:** github.com/agentflare-ai/agentml

| Facet | Value |
|-------|-------|
| enumeration | true, false |

<a id="type-github-com-agentflare-ai-agentml-condlang-datatype"></a>
### 3.4 CondLang.datatype

**Description**
```
Boolean condition expressions (supports In(stateID))
```

- **Kind:** simple
- **Namespace:** github.com/agentflare-ai/agentml


<a id="type-github-com-agentflare-ai-agentml-duration-datatype"></a>
### 3.5 Duration.datatype

**Description**
```
Time duration: 100ms, 5s, 10m, 2h, 3d
```

- **Kind:** simple
- **Namespace:** github.com/agentflare-ai/agentml

| Facet | Value |
|-------|-------|
| pattern | \d*(\.\d+)?(ms|s|m|h|d) |

<a id="type-github-com-agentflare-ai-agentml-encode-datatype"></a>
### 3.6 Encode.datatype

**Description**
```
Data encoding format: "json" (default) or "xml"
```

- **Kind:** simple
- **Namespace:** github.com/agentflare-ai/agentml

| Facet | Value |
|-------|-------|
| enumeration | json, xml |

<a id="type-github-com-agentflare-ai-agentml-eventtype-datatype"></a>
### 3.7 EventType.datatype

**Description**
```
Single event name (e.g., "user.input", "system.error")
```

- **Kind:** simple
- **Namespace:** github.com/agentflare-ai/agentml

| Facet | Value |
|-------|-------|
| pattern | (\i|\d|\-)+(\.(\i|\d|\-)+)* |

<a id="type-github-com-agentflare-ai-agentml-eventtypes-datatype"></a>
### 3.8 EventTypes.datatype

**Description**
```
Event patterns: "user.* system.done" (wildcards and multiple events)
```

- **Kind:** simple
- **Namespace:** github.com/agentflare-ai/agentml

| Facet | Value |
|-------|-------|
| pattern | \.?\*|(\i|\d|\-)+(\.(\i|\d|\-)+)*(\.\*)?(\s(\i|\d|\-)+(\.(\i|\d|\-)+)*(\.\*)?)* |

<a id="type-github-com-agentflare-ai-agentml-exmode-datatype"></a>
### 3.9 Exmode.datatype

**Description**
```
Execution mode: "lax" (permissive) or "strict" (halting on errors)
```

- **Kind:** simple
- **Namespace:** github.com/agentflare-ai/agentml

| Facet | Value |
|-------|-------|
| enumeration | lax, strict |

<a id="type-github-com-agentflare-ai-agentml-historytype-datatype"></a>
### 3.10 HistoryType.datatype

**Description**
```
History depth: "shallow" (immediate substate) or "deep" (full
                configuration)
```

- **Kind:** simple
- **Namespace:** github.com/agentflare-ai/agentml

| Facet | Value |
|-------|-------|
| enumeration | shallow, deep |

<a id="type-github-com-agentflare-ai-agentml-integer-datatype"></a>
### 3.11 Integer.datatype

**Description**
```
Non-negative integer values
```

- **Kind:** simple
- **Namespace:** github.com/agentflare-ai/agentml


<a id="type-github-com-agentflare-ai-agentml-jsonschema-datatype"></a>
### 3.12 JsonSchema.datatype

**Description**
```
JSON Schema reference for validating event data or datamodel
                structures. Supports two formats: 1. Inline JSON: schema='{"type":"object",...}' 2.
                Namespace pointer: schema="prefix:/definitions/TypeName" For inline JSON: CRITICAL
                that the schema includes "description" and "title" fields, ALL properties have
                "description" fields, and uses "required" array for mandatory properties to enable
                proper LLM generation of validation schemas. For namespace pointers: The prefix must
                be declared as schema:prefix attribute on root element. Example: <agentml
                schema:user="file://user-schema.json"> then use schema="user:/definitions/User"
```

- **Kind:** simple
- **Namespace:** github.com/agentflare-ai/agentml

| Facet | Value |
|-------|-------|
| pattern | \{.*\}|[a-zA-Z][a-zA-Z0-9]*:/.* |

<a id="type-github-com-agentflare-ai-agentml-loclang-datatype"></a>
### 3.13 LocLang.datatype

**Description**
```
Datamodel location/path expressions
```

- **Kind:** simple
- **Namespace:** github.com/agentflare-ai/agentml


<a id="type-github-com-agentflare-ai-agentml-transitiontype-datatype"></a>
### 3.14 TransitionType.datatype

**Description**
```
Transition scope: "internal" (same state) or "external" (change
                states)
```

- **Kind:** simple
- **Namespace:** github.com/agentflare-ai/agentml

| Facet | Value |
|-------|-------|
| enumeration | internal, external |

<a id="type-github-com-agentflare-ai-agentml-uri-datatype"></a>
### 3.15 URI.datatype

**Description**
```
URI reference supporting international characters (RFC 3987)
```

- **Kind:** simple
- **Namespace:** github.com/agentflare-ai/agentml


<a id="type-github-com-agentflare-ai-agentml-valuelang-datatype"></a>
### 3.16 ValueLang.datatype

**Description**
```
Value/computation expressions
```

- **Kind:** simple
- **Namespace:** github.com/agentflare-ai/agentml


<a id="type-github-com-agentflare-ai-agentml-agentml-agentml-type"></a>
### 3.17 agentml.agentml.type

- **Kind:** complex
- **Namespace:** github.com/agentflare-ai/agentml
- **Content:** sequence (1 particle(s))


| Attribute | Use | Type | Default | Fixed |
|-----------|-----|------|---------|-------|
| `binding` | optional | [Binding.datatype](#type-github-com-agentflare-ai-agentml-binding-datatype) | — | — |
| `datamodel` | optional | `xs:NMTOKEN` | null | — |
| `initial` | optional | `xs:IDREFS` | — | — |
| `name` | optional | `xs:NMTOKEN` | — | — |
| `version` | required | `xs:decimal` | — | 1.0 |

<a id="type-github-com-agentflare-ai-agentml-agentml-assign-type"></a>
### 3.18 agentml.assign.type

- **Kind:** complex
- **Namespace:** github.com/agentflare-ai/agentml
- **Mixed content:** yes
- **Content:** sequence (1 particle(s))


| Attribute | Use | Type | Default | Fixed |
|-----------|-----|------|---------|-------|
| `attr` | optional | `xs:NMTOKEN` | — | — |
| `expr` | optional | [ValueLang.datatype](#type-github-com-agentflare-ai-agentml-valuelang-datatype) | — | — |
| `location` | required | [LocLang.datatype](#type-github-com-agentflare-ai-agentml-loclang-datatype) | — | — |
| `type` | optional | [AssignType.datatype](#type-github-com-agentflare-ai-agentml-assigntype-datatype) | replacechildren | — |

<a id="type-github-com-agentflare-ai-agentml-agentml-cancel-type"></a>
### 3.19 agentml.cancel.type

- **Kind:** complex
- **Namespace:** github.com/agentflare-ai/agentml
- **Content:** sequence (1 particle(s))


| Attribute | Use | Type | Default | Fixed |
|-----------|-----|------|---------|-------|
| `sendid` | optional | `xs:IDREF` | — | — |
| `sendidexpr` | optional | [ValueLang.datatype](#type-github-com-agentflare-ai-agentml-valuelang-datatype) | — | — |

<a id="type-github-com-agentflare-ai-agentml-agentml-content-type"></a>
### 3.20 agentml.content.type

- **Kind:** complex
- **Namespace:** github.com/agentflare-ai/agentml
- **Mixed content:** yes
- **Content:** sequence (1 particle(s))


| Attribute | Use | Type | Default | Fixed |
|-----------|-----|------|---------|-------|
| `expr` | optional | [ValueLang.datatype](#type-github-com-agentflare-ai-agentml-valuelang-datatype) | — | — |

<a id="type-github-com-agentflare-ai-agentml-agentml-data-type"></a>
### 3.21 agentml.data.type

- **Kind:** complex
- **Namespace:** github.com/agentflare-ai/agentml
- **Mixed content:** yes
- **Content:** sequence (1 particle(s))


| Attribute | Use | Type | Default | Fixed |
|-----------|-----|------|---------|-------|
| `encoding` | optional | [Encode.datatype](#type-github-com-agentflare-ai-agentml-encode-datatype) | json | — |
| `expr` | optional | [ValueLang.datatype](#type-github-com-agentflare-ai-agentml-valuelang-datatype) | — | — |
| `id` | required | `xs:ID` | — | — |
| `schema` | optional | [JsonSchema.datatype](#type-github-com-agentflare-ai-agentml-jsonschema-datatype) | — | — |
| `src` | optional | [URI.datatype](#type-github-com-agentflare-ai-agentml-uri-datatype) | — | — |

<a id="type-github-com-agentflare-ai-agentml-agentml-datamodel-type"></a>
### 3.22 agentml.datamodel.type

- **Kind:** complex
- **Namespace:** github.com/agentflare-ai/agentml
- **Content:** sequence (2 particle(s))


<a id="type-github-com-agentflare-ai-agentml-agentml-donedata-type"></a>
### 3.23 agentml.donedata.type

- **Kind:** complex
- **Namespace:** github.com/agentflare-ai/agentml
- **Content:** choice (2 particle(s))


<a id="type-github-com-agentflare-ai-agentml-agentml-else-type"></a>
### 3.24 agentml.else.type

- **Kind:** complex
- **Namespace:** github.com/agentflare-ai/agentml
- **Content:** sequence (1 particle(s))


<a id="type-github-com-agentflare-ai-agentml-agentml-elseif-type"></a>
### 3.25 agentml.elseif.type

- **Kind:** complex
- **Namespace:** github.com/agentflare-ai/agentml
- **Content:** sequence (1 particle(s))


| Attribute | Use | Type | Default | Fixed |
|-----------|-----|------|---------|-------|
| `cond` | required | [CondLang.datatype](#type-github-com-agentflare-ai-agentml-condlang-datatype) | — | — |

<a id="type-github-com-agentflare-ai-agentml-agentml-final-type"></a>
### 3.26 agentml.final.type

- **Kind:** complex
- **Namespace:** github.com/agentflare-ai/agentml
- **Content:** sequence (1 particle(s))


| Attribute | Use | Type | Default | Fixed |
|-----------|-----|------|---------|-------|
| `id` | optional | `xs:ID` | — | — |

<a id="type-github-com-agentflare-ai-agentml-agentml-finalize-type"></a>
### 3.27 agentml.finalize.type

- **Kind:** complex
- **Namespace:** github.com/agentflare-ai/agentml
- **Content:** sequence (1 particle(s))


<a id="type-github-com-agentflare-ai-agentml-agentml-foreach-type"></a>
### 3.28 agentml.foreach.type

- **Kind:** complex
- **Namespace:** github.com/agentflare-ai/agentml
- **Content:** sequence (1 particle(s))


| Attribute | Use | Type | Default | Fixed |
|-----------|-----|------|---------|-------|
| `array` | required | [ValueLang.datatype](#type-github-com-agentflare-ai-agentml-valuelang-datatype) | — | — |
| `index` | optional | `xs:string` | — | — |
| `item` | required | `xs:string` | — | — |

<a id="type-github-com-agentflare-ai-agentml-agentml-history-type"></a>
### 3.29 agentml.history.type

- **Kind:** complex
- **Namespace:** github.com/agentflare-ai/agentml
- **Content:** sequence (3 particle(s))


| Attribute | Use | Type | Default | Fixed |
|-----------|-----|------|---------|-------|
| `id` | optional | `xs:ID` | — | — |
| `type` | optional | [HistoryType.datatype](#type-github-com-agentflare-ai-agentml-historytype-datatype) | — | — |

<a id="type-github-com-agentflare-ai-agentml-agentml-if-type"></a>
### 3.30 agentml.if.type

- **Kind:** complex
- **Namespace:** github.com/agentflare-ai/agentml
- **Content:** sequence (3 particle(s))


| Attribute | Use | Type | Default | Fixed |
|-----------|-----|------|---------|-------|
| `cond` | required | [CondLang.datatype](#type-github-com-agentflare-ai-agentml-condlang-datatype) | — | — |

<a id="type-github-com-agentflare-ai-agentml-agentml-initial-type"></a>
### 3.31 agentml.initial.type

- **Kind:** complex
- **Namespace:** github.com/agentflare-ai/agentml
- **Content:** sequence (3 particle(s))


<a id="type-github-com-agentflare-ai-agentml-agentml-invoke-type"></a>
### 3.32 agentml.invoke.type

- **Kind:** complex
- **Namespace:** github.com/agentflare-ai/agentml
- **Content:** sequence (1 particle(s))


| Attribute | Use | Type | Default | Fixed |
|-----------|-----|------|---------|-------|
| `autoforward` | optional | [Boolean.datatype](#type-github-com-agentflare-ai-agentml-boolean-datatype) | false | — |
| `id` | optional | `xs:ID` | — | — |
| `idlocation` | optional | [LocLang.datatype](#type-github-com-agentflare-ai-agentml-loclang-datatype) | — | — |
| `namelist` | optional | `xs:string` | — | — |
| `src` | optional | [URI.datatype](#type-github-com-agentflare-ai-agentml-uri-datatype) | — | — |
| `srcexpr` | optional | [ValueLang.datatype](#type-github-com-agentflare-ai-agentml-valuelang-datatype) | — | — |
| `type` | optional | `xs:string` | agentml | — |
| `typeexpr` | optional | [ValueLang.datatype](#type-github-com-agentflare-ai-agentml-valuelang-datatype) | — | — |

<a id="type-github-com-agentflare-ai-agentml-agentml-log-type"></a>
### 3.33 agentml.log.type

- **Kind:** complex
- **Namespace:** github.com/agentflare-ai/agentml
- **Content:** sequence (1 particle(s))


| Attribute | Use | Type | Default | Fixed |
|-----------|-----|------|---------|-------|
| `expr` | optional | [ValueLang.datatype](#type-github-com-agentflare-ai-agentml-valuelang-datatype) | — | — |
| `label` | optional | `xs:string` | — | — |

<a id="type-github-com-agentflare-ai-agentml-agentml-onentry-type"></a>
### 3.34 agentml.onentry.type

- **Kind:** complex
- **Namespace:** github.com/agentflare-ai/agentml
- **Content:** sequence (1 particle(s))


<a id="type-github-com-agentflare-ai-agentml-agentml-onexit-type"></a>
### 3.35 agentml.onexit.type

- **Kind:** complex
- **Namespace:** github.com/agentflare-ai/agentml
- **Content:** sequence (1 particle(s))


<a id="type-github-com-agentflare-ai-agentml-agentml-parallel-type"></a>
### 3.36 agentml.parallel.type

- **Kind:** complex
- **Namespace:** github.com/agentflare-ai/agentml
- **Content:** sequence (1 particle(s))


| Attribute | Use | Type | Default | Fixed |
|-----------|-----|------|---------|-------|
| `id` | optional | `xs:ID` | — | — |

<a id="type-github-com-agentflare-ai-agentml-agentml-param-type"></a>
### 3.37 agentml.param.type

- **Kind:** complex
- **Namespace:** github.com/agentflare-ai/agentml
- **Content:** sequence (1 particle(s))


| Attribute | Use | Type | Default | Fixed |
|-----------|-----|------|---------|-------|
| `expr` | optional | [ValueLang.datatype](#type-github-com-agentflare-ai-agentml-valuelang-datatype) | — | — |
| `location` | optional | [LocLang.datatype](#type-github-com-agentflare-ai-agentml-loclang-datatype) | — | — |
| `name` | required | `xs:NMTOKEN` | — | — |

<a id="type-github-com-agentflare-ai-agentml-agentml-raise-type"></a>
### 3.38 agentml.raise.type

- **Kind:** complex
- **Namespace:** github.com/agentflare-ai/agentml
- **Content:** sequence (1 particle(s))


| Attribute | Use | Type | Default | Fixed |
|-----------|-----|------|---------|-------|
| `event` | required | `xs:NMTOKEN` | — | — |

<a id="type-github-com-agentflare-ai-agentml-agentml-script-type"></a>
### 3.39 agentml.script.type

- **Kind:** complex
- **Namespace:** github.com/agentflare-ai/agentml
- **Mixed content:** yes
- **Content:** sequence (1 particle(s))


| Attribute | Use | Type | Default | Fixed |
|-----------|-----|------|---------|-------|
| `src` | optional | [URI.datatype](#type-github-com-agentflare-ai-agentml-uri-datatype) | — | — |

<a id="type-github-com-agentflare-ai-agentml-agentml-send-type"></a>
### 3.40 agentml.send.type

- **Kind:** complex
- **Namespace:** github.com/agentflare-ai/agentml
- **Content:** sequence (1 particle(s))


| Attribute | Use | Type | Default | Fixed |
|-----------|-----|------|---------|-------|
| `delay` | optional | [Duration.datatype](#type-github-com-agentflare-ai-agentml-duration-datatype) | 0s | — |
| `delayexpr` | optional | [ValueLang.datatype](#type-github-com-agentflare-ai-agentml-valuelang-datatype) | — | — |
| `event` | optional | [EventType.datatype](#type-github-com-agentflare-ai-agentml-eventtype-datatype) | — | — |
| `eventexpr` | optional | [ValueLang.datatype](#type-github-com-agentflare-ai-agentml-valuelang-datatype) | — | — |
| `id` | optional | `xs:ID` | — | — |
| `idlocation` | optional | [LocLang.datatype](#type-github-com-agentflare-ai-agentml-loclang-datatype) | — | — |
| `namelist` | optional | `xs:string` | — | — |
| `target` | optional | [URI.datatype](#type-github-com-agentflare-ai-agentml-uri-datatype) | — | — |
| `targetexpr` | optional | [ValueLang.datatype](#type-github-com-agentflare-ai-agentml-valuelang-datatype) | — | — |
| `type` | optional | `xs:string` | agentml | — |
| `typeexpr` | optional | [ValueLang.datatype](#type-github-com-agentflare-ai-agentml-valuelang-datatype) | — | — |

<a id="type-github-com-agentflare-ai-agentml-agentml-state-type"></a>
### 3.41 agentml.state.type

- **Kind:** complex
- **Namespace:** github.com/agentflare-ai/agentml
- **Content:** sequence (1 particle(s))


| Attribute | Use | Type | Default | Fixed |
|-----------|-----|------|---------|-------|
| `id` | optional | `xs:ID` | — | — |
| `initial` | optional | `xs:IDREFS` | — | — |

<a id="type-github-com-agentflare-ai-agentml-agentml-transition-type"></a>
### 3.42 agentml.transition.type

- **Kind:** complex
- **Namespace:** github.com/agentflare-ai/agentml
- **Content:** sequence (1 particle(s))


| Attribute | Use | Type | Default | Fixed |
|-----------|-----|------|---------|-------|
| `cond` | optional | [CondLang.datatype](#type-github-com-agentflare-ai-agentml-condlang-datatype) | — | — |
| `event` | optional | [EventTypes.datatype](#type-github-com-agentflare-ai-agentml-eventtypes-datatype) | — | — |
| `schema` | optional | [JsonSchema.datatype](#type-github-com-agentflare-ai-agentml-jsonschema-datatype) | — | — |
| `target` | optional | `xs:IDREFS` | — | — |
| `type` | optional | [TransitionType.datatype](#type-github-com-agentflare-ai-agentml-transitiontype-datatype) | — | — |

