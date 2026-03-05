package common

type WithDescription interface {
	Description() string
	SetDescription(value string)
}

type WithDescriptionBase struct {
	DESCRIPTION string `json:"description,omitempty" gorm:"column:description" long:"description" description:"Additional description"`
}

func (d *WithDescriptionBase) Description() string {
	return d.DESCRIPTION
}

func (d *WithDescriptionBase) SetDescription(value string) {
	d.DESCRIPTION = value
}

type WithActive interface {
	IsActive() bool
	SetActive(value bool)
}

type WithActiveBase struct {
	ACTIVE bool `gorm:"index" json:"active" default:"true" long:"active" description:"Active"`
}

func (d *WithActiveBase) IsActive() bool {
	return d.ACTIVE
}

func (d *WithActiveBase) SetActive(value bool) {
	d.ACTIVE = value
}

func (d *WithActiveBase) Init(notActive ...bool) {
	d.ACTIVE = true
	if len(notActive) != 0 {
		if notActive[0] {
			d.ACTIVE = false
		}
	}
}

type WithType interface {
	TypeName() string
	SetTypeName(value string)
}

type WithTypeBase struct {
	TYPE_NAME string `gorm:"index;column:type_name" json:"type_name" validate:"required" vmessage:"Type can not be empty"`
}

func (t *WithTypeBase) TypeName() string {
	return t.TYPE_NAME
}

func (t *WithTypeBase) SetTypeName(value string) {
	t.TYPE_NAME = value
}

type WithRefId interface {
	RefId() string
	SetRefId(value string)
}

type WithRefIdBase struct {
	REFID string `gorm:"index" json:"refid"`
}

func (t *WithRefIdBase) RefId() string {
	return t.REFID
}

func (t *WithRefIdBase) SetRefId(value string) {
	t.REFID = value
}

type WithLongName interface {
	LongName() string
	SetLongName(name string)
}

type WithLongNameBase struct {
	LONG_NAME string `json:"long_name,omitempty"`
}

func (t *WithLongNameBase) LongName() string {
	return t.LONG_NAME
}

func (t *WithLongNameBase) SetLongName(value string) {
	t.LONG_NAME = value
}

type WithUniqueName interface {
	WithName
}

type WithUniqueNameBase struct {
	NAME string `gorm:"uniqueIndex" json:"name" validate:"required" vmessage:"Name can not be empty" long:"name" description:"Unique name"`
}

func (w *WithUniqueNameBase) Name() string {
	return w.NAME
}

func (w *WithUniqueNameBase) SetName(name string) {
	w.NAME = name
}

type WithTitle interface {
	Title() string
	SetTitle(value string)
}

type WithTitleBase struct {
	TITLE string `json:"title,omitempty" gorm:"column:title" long:"title" description:"Title, e.g. job position"`
}

func (d *WithTitleBase) Title() string {
	return d.TITLE
}

func (d *WithTitleBase) SetTitle(value string) {
	d.TITLE = value
}
