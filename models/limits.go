package models

import "github.com/doveccl/doj/contract/limits"

const (
	UserNameMin = 3
	UserNameMax = 32

	LanguageIDMax = 32
	StatusMax     = 32

	KindMax = 8
	ModeMax = 16
	SortMax = 16

	NameMax       = 64
	SettingKeyMax = 64

	AuthMax   = 128
	SourceMax = 128

	MailMax  = 256
	TitleMax = 256
	BioMax   = 256

	AvatarMax      = 512
	CaseMessageMax = limits.MaxCaseMessageRunes
)
