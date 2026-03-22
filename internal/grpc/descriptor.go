package grpc

import (
	"encoding/json"
	"fmt"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/desc/protoparse"
	"github.com/jhump/protoreflect/dynamic"
)

// Registry compiles .proto files at startup and provides method/message lookup
// and JSON ↔ proto transcoding without requiring protoc code generation.
type Registry struct {
	methods map[string]*desc.MethodDescriptor // key: "/package.Service/Method"
	factory *dynamic.MessageFactory
}

// NewRegistry parses the given .proto files and builds a method registry.
// importPaths are directories searched when resolving proto imports.
func NewRegistry(protoFiles []string, importPaths []string) (*Registry, error) {
	if len(protoFiles) == 0 {
		return &Registry{
			methods: make(map[string]*desc.MethodDescriptor),
			factory: dynamic.NewMessageFactoryWithDefaults(),
		}, nil
	}

	p := protoparse.Parser{
		ImportPaths:           importPaths,
		InferImportPaths:      true,
		IncludeSourceCodeInfo: false,
	}

	fds, err := p.ParseFiles(protoFiles...)
	if err != nil {
		return nil, fmt.Errorf("parsing proto files: %w", err)
	}

	methods := make(map[string]*desc.MethodDescriptor)
	for _, fd := range fds {
		for _, svc := range fd.GetServices() {
			for _, m := range svc.GetMethods() {
				key := fmt.Sprintf("/%s/%s", svc.GetFullyQualifiedName(), m.GetName())
				methods[key] = m
			}
		}
	}

	return &Registry{
		methods: methods,
		factory: dynamic.NewMessageFactoryWithDefaults(),
	}, nil
}

// FindMethod returns the MethodDescriptor for the given full gRPC method path.
// Returns nil, nil when the method is not in any loaded .proto file.
func (r *Registry) FindMethod(fullMethod string) (*desc.MethodDescriptor, error) {
	md, ok := r.methods[fullMethod]
	if !ok {
		return nil, nil
	}
	return md, nil
}

// Methods returns all registered full method paths.
func (r *Registry) Methods() []string {
	out := make([]string, 0, len(r.methods))
	for k := range r.methods {
		out = append(out, k)
	}
	return out
}

// DecodeRequest deserialises the raw protobuf bytes of a request message into
// a map[string]interface{} using protojson field names, ready for condition
// evaluation (the same evalCondition / extractValue logic used for REST).
func (r *Registry) DecodeRequest(md *desc.MethodDescriptor, b []byte) (map[string]interface{}, error) {
	msg := r.factory.NewDynamicMessage(md.GetInputType())
	if err := msg.Unmarshal(b); err != nil {
		return nil, fmt.Errorf("unmarshal request: %w", err)
	}

	jsonBytes, err := msg.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("marshal request to json: %w", err)
	}

	var out map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &out); err != nil {
		return nil, fmt.Errorf("parse request json: %w", err)
	}
	return out, nil
}

// EncodeResponse takes a JSON blob (protojson-compatible) and serialises it
// into wire bytes for the given method's output type.
func (r *Registry) EncodeResponse(md *desc.MethodDescriptor, jsonBytes []byte) ([]byte, error) {
	msg := r.factory.NewDynamicMessage(md.GetOutputType())
	if err := msg.UnmarshalJSON(jsonBytes); err != nil {
		return nil, fmt.Errorf("unmarshal response json: %w", err)
	}
	b, err := msg.Marshal()
	if err != nil {
		return nil, fmt.Errorf("marshal response proto: %w", err)
	}
	return b, nil
}
