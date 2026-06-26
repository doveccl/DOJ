package utils

const (
	BodyLimitShortText    = "256K"
	BodyLimitMarkdown     = "2M"
	BodyLimitSource       = "1M"
	BodyLimitLanguage     = "128K"
	BodyLimitSettings     = "2M"
	BodyLimitImage        = "6M"
	BodyLimitAsset        = "130M"
	BodyLimitEditAsset    = "2M"
	BodyLimitJudgerResult = "2M"
)

const (
	MaxMarkdownBytes        = 2 << 20
	MaxSourceBytes          = 1 << 20
	MaxShortTextBytes       = 256 << 10
	MaxLanguageCommandBytes = 128 << 10
	MaxJudgerMessageBytes   = 256 << 10
)
