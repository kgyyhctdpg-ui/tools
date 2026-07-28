package math

type OMath struct {
	CT_OMath
	EG_OMathMathElements []*EG_OMathMathElements
}

type CT_OMath struct{}

type OMathPara struct {
	OMath []*OMath
}

type EG_OMathMathElements struct {
	R       *CT_R
	F       *CT_F
	Rad     *CT_Rad
	SSub    *CT_SSub
	SSup    *CT_SSup
	SSubSup *CT_SSubSup
	Nary    *CT_Nary
	LimLow  *CT_LimLow
	LimUpp  *CT_LimUpp
	Acc     *CT_Acc
	D       *CT_D
	M       *CT_M
}

type CT_OMathArg struct {
	EG_OMathMathElements []*EG_OMathMathElements
}

type CT_R struct {
	Choice []*CT_RChoice
}

type CT_RChoice struct {
	T []*CT_Text
}

type CT_Text struct {
	Content string
}

type CT_Char struct {
	ValAttr string
}

type CT_F struct {
	Num *CT_OMathArg
	Den *CT_OMathArg
	FPr *CT_FPr
}

type CT_FPr struct {
	Type *CT_FType
}

type CT_FType struct {
	ValAttr string
}

type CT_Rad struct {
	E     *CT_OMathArg
	Deg   *CT_OMathArg
	RadPr *CT_RadPr
}

type CT_RadPr struct {
	DegHide *CT_OnOff
}

type CT_OnOff struct{}

type CT_SSub struct {
	E      *CT_OMathArg
	Sub    *CT_OMathArg
	SSubPr *CT_SSubPr
}

type CT_SSubPr struct{}

type CT_SSup struct {
	E      *CT_OMathArg
	Sup    *CT_OMathArg
	SSupPr *CT_SSupPr
}

type CT_SSupPr struct{}

type CT_SSubSup struct {
	E         *CT_OMathArg
	Sub       *CT_OMathArg
	Sup       *CT_OMathArg
	SSubSupPr *CT_SSubSupPr
}

type CT_SSubSupPr struct{}

type CT_Nary struct {
	NaryPr *CT_NaryPr
	Sub    *CT_OMathArg
	Sup    *CT_OMathArg
	E      *CT_OMathArg
}

type CT_NaryPr struct {
	Chr *CT_Char
}

type CT_LimLow struct {
	E   *CT_OMathArg
	Lim *CT_OMathArg
}

type CT_LimUpp struct {
	E   *CT_OMathArg
	Lim *CT_OMathArg
}

type CT_Acc struct {
	AccPr *CT_AccPr
	E     *CT_OMathArg
}

type CT_AccPr struct {
	Chr *CT_Char
}

type CT_D struct {
	DPr *CT_DPr
	E   []*CT_OMathArg
}

type CT_DPr struct {
	BegChr *CT_Char
	EndChr *CT_Char
}

type CT_M struct {
	MPr *CT_MPr
	Mr  []*CT_MR
}

type CT_MPr struct{}

type CT_MR struct {
	E []*CT_OMathArg
}

const ST_FTypeNoBar = "noBar"

func NewOMath() *OMath         { return &OMath{} }
func NewOMathPara() *OMathPara { return &OMathPara{} }
func NewEG_OMathMathElements() *EG_OMathMathElements {
	return &EG_OMathMathElements{}
}
func NewCT_OMathArg() *CT_OMathArg   { return &CT_OMathArg{} }
func NewCT_R() *CT_R                 { return &CT_R{} }
func NewCT_Text() *CT_Text           { return &CT_Text{} }
func NewCT_Char() *CT_Char           { return &CT_Char{} }
func NewCT_F() *CT_F                 { return &CT_F{} }
func NewCT_FPr() *CT_FPr             { return &CT_FPr{} }
func NewCT_FType() *CT_FType         { return &CT_FType{} }
func NewCT_Rad() *CT_Rad             { return &CT_Rad{} }
func NewCT_RadPr() *CT_RadPr         { return &CT_RadPr{} }
func NewCT_OnOff() *CT_OnOff         { return &CT_OnOff{} }
func NewCT_SSub() *CT_SSub           { return &CT_SSub{} }
func NewCT_SSubPr() *CT_SSubPr       { return &CT_SSubPr{} }
func NewCT_SSup() *CT_SSup           { return &CT_SSup{} }
func NewCT_SSupPr() *CT_SSupPr       { return &CT_SSupPr{} }
func NewCT_SSubSup() *CT_SSubSup     { return &CT_SSubSup{} }
func NewCT_SSubSupPr() *CT_SSubSupPr { return &CT_SSubSupPr{} }
func NewCT_Nary() *CT_Nary           { return &CT_Nary{} }
func NewCT_NaryPr() *CT_NaryPr       { return &CT_NaryPr{} }
func NewCT_LimLow() *CT_LimLow       { return &CT_LimLow{} }
func NewCT_LimUpp() *CT_LimUpp       { return &CT_LimUpp{} }
func NewCT_Acc() *CT_Acc             { return &CT_Acc{} }
func NewCT_AccPr() *CT_AccPr         { return &CT_AccPr{} }
func NewCT_D() *CT_D                 { return &CT_D{} }
func NewCT_DPr() *CT_DPr             { return &CT_DPr{} }
func NewCT_M() *CT_M                 { return &CT_M{} }
func NewCT_MPr() *CT_MPr             { return &CT_MPr{} }
func NewCT_MR() *CT_MR               { return &CT_MR{} }
