package types

import "time"

// PropertyType is the data type of a single property in a [TypeDefinition].
//
// Primitive types map directly to Go/JSON primitives.
// Choice types (option, select, multiselect) declare the field shape; the
// allowed option values are stored as runtime data in the service's own
// collection, not baked into the schema definition.
// Complex types carry additional configuration on [PropertyDefinition].
type PropertyType string

const (
	// Primitive types.

	// PropertyTypeString is a UTF-8 text value.
	PropertyTypeString PropertyType = "string"

	// PropertyTypeInteger is a 64-bit signed integer.
	PropertyTypeInteger PropertyType = "integer"

	// PropertyTypeFloat is a 64-bit floating-point number.
	PropertyTypeFloat PropertyType = "float"

	// PropertyTypeDate is an ISO 8601 date (e.g. "2026-01-15").
	PropertyTypeDate PropertyType = "date"

	// PropertyTypeDatetime is an ISO 8601 date + time (e.g. "2026-01-15T10:30:00Z").
	PropertyTypeDatetime PropertyType = "datetime"

	// PropertyTypeBoolean is a true/false flag.
	PropertyTypeBoolean PropertyType = "boolean"

	// Choice types — allowed values are stored as runtime data, not in the schema.

	// PropertyTypeOption is a single fixed value from a predefined set.
	PropertyTypeOption PropertyType = "option"

	// PropertyTypeSelect is a single value chosen from a list at runtime.
	PropertyTypeSelect PropertyType = "select"

	// PropertyTypeMultiSelect is one or more values chosen from a list at runtime.
	PropertyTypeMultiSelect PropertyType = "multiselect"

	// Complex types — require additional configuration on [PropertyDefinition].

	// PropertyTypeRating is a numeric rating with a configurable range and labels.
	// A non-nil [RatingConfig] must be supplied on the [PropertyDefinition].
	PropertyTypeRating PropertyType = "rating"
)

// RatingConfig holds the configuration for a property of type [PropertyTypeRating].
// The Agency Owner provides Min, Max, and optionally Labels when defining the property.
type RatingConfig struct {
	// Min is the lowest allowed rating value (e.g. 1).
	Min int

	// Max is the highest allowed rating value (e.g. 5).
	Max int

	// Labels are optional human-readable names for each value from Min to Max.
	// If provided, len(Labels) must equal Max - Min + 1.
	Labels []string
}

// PropertyDefinition describes a single named property within a [TypeDefinition].
type PropertyDefinition struct {
	// Name is the property identifier (e.g. "pressure", "status", "rating").
	Name string

	// Type is the data type for this property.
	Type PropertyType

	// Required indicates that every instance of this type must supply this property.
	Required bool

	// RatingConfig holds the configuration for rating properties.
	// Must be non-nil when Type is [PropertyTypeRating]; ignored for all other types.
	RatingConfig *RatingConfig
}

// TypeDefinition declares a named class of entity within a [Schema].
//
// In the Digital Twin service a type is a real-world entity class
// (e.g. "Pump", "Pipe", "Person"). In the Comm service a type is a participant
// or message class (e.g. "Channel", "Subscriber"). The same structure is used
// in both services; semantics differ by context.
type TypeDefinition struct {
	// Name is the unique type identifier within the Schema (e.g. "Pump").
	Name string

	// Description provides human-readable context for this type.
	Description string

	// Properties is the ordered list of property definitions for this type.
	Properties []PropertyDefinition
}

// Schema is a versioned, immutable collection of [TypeDefinition]s for one
// service within one agency. Each service (DT, Comm) maintains its own
// independent Schema per agency. Updating the schema produces a new version;
// previous versions are preserved.
type Schema struct {
	// ID is the unique identifier for this schema version (UUID).
	ID string

	// AgencyID is the agency this schema belongs to.
	AgencyID string

	// Version is the auto-incrementing version number (1, 2, 3, …).
	// The first publish produces Version 1; each subsequent call increments by one.
	Version int

	// Tag is the human-readable version label (e.g. "v1", "v2").
	Tag string

	// Types is the ordered list of type definitions in this schema version.
	Types []TypeDefinition

	// CreatedAt is the time this schema version was created.
	CreatedAt time.Time
}
