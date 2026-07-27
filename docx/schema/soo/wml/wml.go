package wml

type ST_Jc int

const (
	ST_JcLeft ST_Jc = iota
	ST_JcCenter
	ST_JcRight
	ST_JcBoth
)

const ST_JcTableCenter ST_Jc = ST_JcCenter

type ST_Border int

const ST_BorderSingle ST_Border = iota

type ST_Merge int

const (
	ST_MergeUnset ST_Merge = iota
	ST_MergeRestart
	ST_MergeContinue
)

type ST_VerticalJc int

const ST_VerticalJcCenter ST_VerticalJc = iota

type ST_Shd int

const ST_ShdClear ST_Shd = iota

type ST_StyleType int

const ST_StyleTypeCharacter ST_StyleType = iota

type ST_TabJc int

const ST_TabJcRight ST_TabJc = iota

type ST_TabTlc int

const ST_TabTlcDot ST_TabTlc = iota

type ST_LineSpacingRule int

const ST_LineSpacingRuleAuto ST_LineSpacingRule = iota

type ST_HdrFtr int

const ST_HdrFtrDefault ST_HdrFtr = iota

type ST_SectionMark int

const (
	ST_SectionMarkUnset ST_SectionMark = iota
	ST_SectionMarkNextPage
)

type ST_PageOrientation int

const (
	ST_PageOrientationPortrait ST_PageOrientation = iota
	ST_PageOrientationLandscape
)

type ST_NumberFormat int

const (
	ST_NumberFormatBullet ST_NumberFormat = iota
	ST_NumberFormatDecimal
)

type ST_SignedTwipsMeasure struct {
	Int64 *int64
}

type CT_Tabs struct {
	Tab []*CT_TabStop
}

func NewCT_Tabs() *CT_Tabs {
	return &CT_Tabs{}
}

type CT_TabStop struct {
	ValAttr    ST_TabJc
	LeaderAttr ST_TabTlc
	PosAttr    ST_SignedTwipsMeasure
}

func NewCT_TabStop() *CT_TabStop {
	return &CT_TabStop{}
}

type CT_Ind struct {
	RightAttr *ST_SignedTwipsMeasure
}

func NewCT_Ind() *CT_Ind {
	return &CT_Ind{}
}

type CT_PPr struct {
	Tabs   *CT_Tabs
	Ind    *CT_Ind
	SectPr *CT_SectPr
}

type CT_TcPr struct {
	GridSpan *CT_DecimalNumber
}

type CT_SectPr struct {
	PgSz                *CT_PageSz
	EG_HdrFtrReferences []*EG_HdrFtrReferences
}

func NewCT_SectPr() *CT_SectPr {
	return &CT_SectPr{}
}

type CT_PageSz struct {
	OrientAttr ST_PageOrientation
	WAttr      any
	HAttr      any
}

func NewCT_PageSz() *CT_PageSz {
	return &CT_PageSz{}
}

type CT_DecimalNumber struct {
	ValAttr int64
}

type EG_HdrFtrReferences struct {
	HeaderReference *CT_HdrFtrRef
	FooterReference *CT_HdrFtrRef
}

type CT_HdrFtrRef struct {
	TypeAttr ST_HdrFtr
}

type CT_P struct {
	EG_PContent []*EG_PContent
}

type EG_PContent struct {
	EG_ContentRunContent []*EG_ContentRunContent
}

func NewEG_PContent() *EG_PContent {
	return &EG_PContent{}
}

type EG_ContentRunContent struct {
	EG_RunLevelElts []*EG_RunLevelElts
}

func NewEG_ContentRunContent() *EG_ContentRunContent {
	return &EG_ContentRunContent{}
}

type EG_RunLevelElts struct {
	EG_MathContent []*EG_MathContent
}

func NewEG_RunLevelElts() *EG_RunLevelElts {
	return &EG_RunLevelElts{}
}

type EG_MathContent struct {
	OMath     any
	OMathPara any
}

func NewEG_MathContent() *EG_MathContent {
	return &EG_MathContent{}
}
