package mapper

import "fmt"

type Registry struct {
	workflows   map[WorkflowName]BlockWorkflow
	valueCodecs map[ValueCodecName]ValueCodec
	blockCodecs map[BlockCodecName]BlockCodec
	tagHandlers map[string]map[string]TagHandler
	built       bool
}

func NewRegistry() *Registry {
	return &Registry{
		workflows:   builtinWorkflows(),
		valueCodecs: map[ValueCodecName]ValueCodec{ValueCodecBuiltin: builtinValueCodec{}},
		blockCodecs: make(map[BlockCodecName]BlockCodec),
		tagHandlers: make(map[string]map[string]TagHandler),
	}
}

// RegisterTagHandler registers one named implementation for an application-owned
// struct tag. The excel tag is reserved by the mapper.
func (r *Registry) RegisterTagHandler(tag, name string, handler TagHandler) error {
	if r == nil {
		return fmt.Errorf("%w: nil registry", ErrInvalidRegistry)
	}
	if r.built {
		return fmt.Errorf("%w: registry is already built", ErrInvalidRegistry)
	}
	if err := validateExtensionTagName(tag); err != nil {
		return err
	}
	if err := validateTagHandlerName(name); err != nil {
		return err
	}
	if handler == nil {
		return fmt.Errorf("%w: invalid tag handler registration", ErrInvalidRegistry)
	}
	if r.tagHandlers == nil {
		r.tagHandlers = make(map[string]map[string]TagHandler)
	}
	handlers := r.tagHandlers[tag]
	if handlers == nil {
		handlers = make(map[string]TagHandler)
		r.tagHandlers[tag] = handlers
	}
	if _, exists := handlers[name]; exists {
		return fmt.Errorf("%w: tag handler %q for %q is already registered", ErrInvalidRegistry, name, tag)
	}
	handlers[name] = handler
	return nil
}

func (r *Registry) RegisterBlockCodec(name BlockCodecName, codec BlockCodec) error {
	if r == nil {
		return fmt.Errorf("%w: nil registry", ErrInvalidRegistry)
	}
	if r.built {
		return fmt.Errorf("%w: registry is already built", ErrInvalidRegistry)
	}
	if name == "" || codec == nil {
		return fmt.Errorf("%w: invalid block codec registration", ErrInvalidRegistry)
	}
	if r.blockCodecs == nil {
		r.blockCodecs = make(map[BlockCodecName]BlockCodec)
	}
	if _, exists := r.blockCodecs[name]; exists {
		return fmt.Errorf("%w: block codec %q is already registered", ErrInvalidRegistry, name)
	}
	r.blockCodecs[name] = codec
	return nil
}

func (r *Registry) RegisterBlockWorkflow(name WorkflowName, workflow BlockWorkflow) error {
	if r == nil {
		return fmt.Errorf("%w: nil registry", ErrInvalidRegistry)
	}
	if r.built {
		return fmt.Errorf("%w: registry is already built", ErrInvalidRegistry)
	}
	if name == "" || workflow == nil {
		return fmt.Errorf("%w: invalid workflow registration", ErrInvalidRegistry)
	}
	if r.workflows == nil {
		r.workflows = make(map[WorkflowName]BlockWorkflow)
	}
	if _, exists := r.workflows[name]; exists {
		return fmt.Errorf("%w: workflow %q is already registered", ErrInvalidRegistry, name)
	}
	r.workflows[name] = workflow
	return nil
}

func (r *Registry) RegisterValueCodec(name ValueCodecName, codec ValueCodec) error {
	if r == nil {
		return fmt.Errorf("%w: nil registry", ErrInvalidRegistry)
	}
	if r.built {
		return fmt.Errorf("%w: registry is already built", ErrInvalidRegistry)
	}
	if name == "" || codec == nil {
		return fmt.Errorf("%w: invalid value codec registration", ErrInvalidRegistry)
	}
	if r.valueCodecs == nil {
		r.valueCodecs = make(map[ValueCodecName]ValueCodec)
	}
	if _, exists := r.valueCodecs[name]; exists {
		return fmt.Errorf("%w: value codec %q is already registered", ErrInvalidRegistry, name)
	}
	r.valueCodecs[name] = codec
	return nil
}

func (r *Registry) Build() *Registry {
	if r == nil {
		return defaultRegistry()
	}
	copyRegistry := &Registry{
		workflows:   make(map[WorkflowName]BlockWorkflow, len(r.workflows)),
		valueCodecs: make(map[ValueCodecName]ValueCodec, len(r.valueCodecs)),
		blockCodecs: make(map[BlockCodecName]BlockCodec, len(r.blockCodecs)),
		tagHandlers: make(map[string]map[string]TagHandler, len(r.tagHandlers)),
		built:       true,
	}
	for name, value := range r.workflows {
		copyRegistry.workflows[name] = value
	}
	for name, value := range r.valueCodecs {
		copyRegistry.valueCodecs[name] = value
	}
	for name, value := range r.blockCodecs {
		copyRegistry.blockCodecs[name] = value
	}
	for tag, handlers := range r.tagHandlers {
		copied := make(map[string]TagHandler, len(handlers))
		for name, handler := range handlers {
			copied[name] = handler
		}
		copyRegistry.tagHandlers[tag] = copied
	}
	return copyRegistry
}

func defaultRegistry() *Registry { return NewRegistry().Build() }
