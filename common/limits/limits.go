package limits

const (
	BodyShortText    = "256K"
	BodyMarkdown     = "2M"
	BodySource       = "1M"
	BodyLanguage     = "128K"
	BodySettings     = "2M"
	BodyImage        = "6M"
	BodyAsset        = "130M"
	BodyEditAsset    = "2M"
	BodyJudgerResult = "2M"
)

const (
	MaxMarkdownBytes        = 2 << 20
	MaxSourceBytes          = 1 << 20
	MaxShortTextBytes       = 256 << 10
	MaxLanguageCommandBytes = 128 << 10
	MaxJudgerMessageBytes   = 256 << 10
)
