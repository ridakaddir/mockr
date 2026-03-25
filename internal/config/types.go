package config

// Config is the top-level structure for mockr config files (JSON/YAML/TOML).
type Config struct {
	Routes     []Route     `json:"routes"      yaml:"routes"      toml:"routes"`
	GRPCRoutes []GRPCRoute `json:"grpc_routes" yaml:"grpc_routes" toml:"grpc_routes"`
}

// GRPCRoute defines a single interceptable gRPC method.
// Match is the full gRPC method path: "/package.Service/Method".
// Cases reuse the same Case struct; Case.Status maps to a gRPC status code
// (0=OK, 1=CANCELLED, 2=UNKNOWN, 3=INVALID_ARGUMENT, 4=DEADLINE_EXCEEDED,
//
//	5=NOT_FOUND, 6=ALREADY_EXISTS, 7=PERMISSION_DENIED, 13=INTERNAL, 14=UNAVAILABLE).
//
// Case.File and Case.JSON hold protojson-compatible JSON (field names match the proto field names).
type GRPCRoute struct {
	Match       string          `json:"match"       yaml:"match"       toml:"match"`
	Enabled     *bool           `json:"enabled"     yaml:"enabled"     toml:"enabled"`
	Fallback    string          `json:"fallback"    yaml:"fallback"    toml:"fallback"`
	Conditions  []Condition     `json:"conditions"  yaml:"conditions"  toml:"conditions"`
	Cases       map[string]Case `json:"cases"       yaml:"cases"       toml:"cases"`
	Transitions []Transition    `json:"transitions" yaml:"transitions" toml:"transitions"`
}

// IsEnabled returns true if the gRPC route is enabled (defaults to true if not set).
func (r *GRPCRoute) IsEnabled() bool {
	if r.Enabled == nil {
		return true
	}
	return *r.Enabled
}

// Route defines a single interceptable HTTP endpoint.
type Route struct {
	Method      string          `json:"method"      yaml:"method"      toml:"method"`
	Match       string          `json:"match"       yaml:"match"       toml:"match"`
	Enabled     *bool           `json:"enabled"     yaml:"enabled"     toml:"enabled"`
	Fallback    string          `json:"fallback"    yaml:"fallback"    toml:"fallback"`
	Conditions  []Condition     `json:"conditions"  yaml:"conditions"  toml:"conditions"`
	Cases       map[string]Case `json:"cases"       yaml:"cases"       toml:"cases"`
	Transitions []Transition    `json:"transitions" yaml:"transitions" toml:"transitions"`
}

// Transition defines one step in a time-based response sequence.
type Transition struct {
	Case     string `json:"case"     yaml:"case"     toml:"case"`
	Duration int    `json:"duration" yaml:"duration" toml:"duration"` // seconds this state lasts; omit or 0 on the last entry for a terminal state
}

// IsEnabled returns true if the route is enabled (defaults to true if not set).
func (r *Route) IsEnabled() bool {
	if r.Enabled == nil {
		return true
	}
	return *r.Enabled
}

// Condition maps an incoming request attribute to a case name.
type Condition struct {
	Source string `json:"source" yaml:"source" toml:"source"` // body | query | header
	Field  string `json:"field"  yaml:"field"  toml:"field"`  // dot-notation or key name
	Op     string `json:"op"     yaml:"op"     toml:"op"`     // eq | neq | contains | regex | exists | not_exists
	Value  string `json:"value"  yaml:"value"  toml:"value"`
	Case   string `json:"case"   yaml:"case"   toml:"case"` // case key to activate
}

// Case defines a mock response.
type Case struct {
	Status   int    `json:"status"    yaml:"status"    toml:"status"`
	JSON     string `json:"json"      yaml:"json"      toml:"json"`
	File     string `json:"file"      yaml:"file"      toml:"file"`
	Delay    int    `json:"delay"     yaml:"delay"     toml:"delay"`
	Persist  bool   `json:"persist"   yaml:"persist"   toml:"persist"`
	Merge    string `json:"merge"     yaml:"merge"     toml:"merge"`     // append | update | delete
	Key      string `json:"key"       yaml:"key"       toml:"key"`       // record lookup key
	ArrayKey string `json:"array_key" yaml:"array_key" toml:"array_key"` // array field in stub JSON
	Defaults string `json:"defaults"  yaml:"defaults"  toml:"defaults"`  // JSON file with default values for append/update
}

// StatusCode returns the HTTP status for a case, defaulting to 200.
func (c *Case) StatusCode() int {
	if c.Status == 0 {
		return 200
	}
	return c.Status
}
