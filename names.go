package mapper

// WorkflowName identifies a registered block workflow in an excel tag.
type WorkflowName string

const (
	WorkflowAll         WorkflowName = "all"
	WorkflowIndex       WorkflowName = "index"
	WorkflowStart       WorkflowName = "start"
	WorkflowTitle       WorkflowName = "title"
	WorkflowRepeatTitle WorkflowName = "repeat_title"
	WorkflowTitleRange  WorkflowName = "title_range"
)

// BlockFormat identifies a built-in block representation.
type BlockFormat string

const (
	FormatStruct    BlockFormat = "struct"
	FormatSlice     BlockFormat = "slice"
	FormatSingle    BlockFormat = "single"
	FormatGroup     BlockFormat = "group"
	FormatForm      BlockFormat = "form"
	FormatTranspose BlockFormat = "transpose"
)

// ValueCodecName identifies a registered value codec in an excel tag.
type ValueCodecName string

const ValueCodecBuiltin ValueCodecName = "builtin"

// BlockCodecName identifies a registered whole-block codec in an excel tag.
type BlockCodecName string

const (
	tagNameExcel          = "excel"
	tagKeyKey             = "key"
	tagKeyWorkflow        = "workflow"
	tagKeyFormat          = "format"
	tagKeyStartRow        = "start_row"
	tagKeyEndRow          = "end_row"
	tagKeyTitle           = "title"
	tagKeyEndTitle        = "end_title"
	tagKeyMinRows         = "min_rows"
	tagKeyBlankLine       = "blank_line"
	tagKeyOptional        = "optional"
	tagKeyBlockCodec      = "block_codec"
	tagKeyCodecOptions    = "codec_options"
	tagKeyIncludeEndBlock = "include_end_block"
	tagKeyDataRow         = "data_row"
	tagKeyLabelCol        = "label_col"
	tagKeyValueCol        = "value_col"
	tagKeyLabel           = "label"
	tagKeyHeader          = "header"
	tagKeyRequired        = "required"
	tagKeyAllowEmpty      = "allow_empty"
	tagKeySkipDecode      = "skip_decode"
	tagKeySkipEncode      = "skip_encode"
	tagKeyMultiCell       = "multi_cell"
	tagKeySeparator       = "separator"
	tagKeyValueCodec      = "value_codec"
	tagKeyValidate        = "validate"
)
