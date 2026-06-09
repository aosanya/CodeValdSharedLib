package types

import "strings"

// TopicsFromSchema derives the standard pub/sub topic list for a service from
// its schema. For each [TypeDefinition] with [TypeDefinition.PublishEvents] set
// to true it emits:
//
//   - {prefix}.{type}.created               — always
//   - {prefix}.{type}.updated               — skipped when Immutable
//   - {prefix}.{type}.update.{field}        — one per property, skipped when Immutable
//   - {prefix}.{type}.deleted               — always
//
// The type segment is strings.ToLower(TypeDefinition.Name), e.g. "Task" →
// "work.task.created", "work.task.update.status". Topic order follows schema
// definition order; properties follow their declaration order within each type.
//
// When servicePrefix is empty the leading "{prefix}." segment is omitted so
// callers participating in the intent-keyed naming rename (BUG-20260609-001)
// can pass "" without producing a leading dot.
//
// Call this from a service's AllTopics function so the registrar's produces
// list stays in sync with the schema without manual maintenance.
func TopicsFromSchema(servicePrefix string, schema Schema) []string {
	var topics []string
	for _, td := range schema.Types {
		if !td.PublishEvents {
			continue
		}
		base := strings.ToLower(td.Name) + "."
		if servicePrefix != "" {
			base = servicePrefix + "." + base
		}
		topics = append(topics, base+"created")
		if !td.Immutable {
			topics = append(topics, base+"updated")
			for _, p := range td.Properties {
				topics = append(topics, base+"update."+p.Name)
			}
		}
		topics = append(topics, base+"deleted")
	}
	return topics
}
