package xlsx

type Option func(*config)

type config struct {
	sheetName     string
	date1904      bool
	allowFormulas bool
}

func WithSheetName(name string) Option { return func(config *config) { config.sheetName = name } }
func WithDate1904(enabled bool) Option { return func(config *config) { config.date1904 = enabled } }
func WithFormulas(enabled bool) Option {
	return func(config *config) { config.allowFormulas = enabled }
}

func configure(options []Option) config {
	var result config
	for _, option := range options {
		if option != nil {
			option(&result)
		}
	}
	return result
}
