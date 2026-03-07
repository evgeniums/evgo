package common

import (
	"strings"

	"github.com/evgeniums/evgo/pkg/utils"
)

type WithName interface {
	Name() string
	SetName(name string)
}

func ConstructPath(sections []string, separator string) string {
	return strings.Join(sections, separator)
}

// Interface for type having name and path.
type WithPath interface {
	Path() string
	FullPath() string
	Paths() []string
	Sections() []string
	Separator() string

	SetParent(parent WithPath)
}

type WithNameAndPath interface {
	WithName
	WithPath
}

type WithPathParent interface {
	WithPath
	AddChild(child WithPath)
}

type WithNameBaseConfig struct {
	NAME string `gorm:"index" json:"name,omitempty" long:"name" description:"Name"`
}

func (e *WithNameBaseConfig) Init(name string) {
	e.NAME = name
}

func (w *WithNameBaseConfig) Name() string {
	return w.NAME
}

func (w *WithNameBaseConfig) SetName(name string) {
	w.NAME = name
}

type WithNameBase = WithNameBaseConfig

// Base type for types having name and path.
type WithPathBase struct {
	path      string
	sections  []string
	separator string
}

func adjustFirstSection(sections []string) []string {
	if len(sections) > 0 && sections[0] == "" {
		return sections[1:]
	}
	return sections
}

func (w *WithPathBase) Init(path string, separator ...string) {
	w.path = path
	w.separator = utils.OptionalArg("/", separator...)
	w.sections = strings.Split(w.path, w.separator)
}

func (w *WithPathBase) Path() string {
	return w.path
}

func (w *WithPathBase) Separator() string {
	return w.separator
}

func (w *WithPathBase) Sections() []string {
	return w.sections
}

func (w *WithPathBase) Paths() []string {
	paths := []string{"/"}
	for i := 1; i < len(w.sections); i++ {
		paths = append(paths, ConstructPath(w.sections[0:i+1], w.separator))
	}
	return paths
}

func (w *WithPathBase) FullPath() string {
	return ConstructPath(w.sections, w.separator)
}

func (w *WithPathBase) SetParent(parent WithPath) {
	w.sections = strings.Split(w.path, w.separator)
	w.sections = adjustFirstSection(w.sections)
	ps := parent.Sections()
	last := ps[len(ps)-1]
	if last == "" {
		w.sections = append(ps[:len(ps)-1], w.sections[0:]...)
	} else {
		w.sections = append(ps, w.sections[0:]...)
	}
}

// Base type for types having name and path and children.
type WithPathParentBase struct {
	WithPathBase
}

func (w *WithPathParentBase) AddChild(child WithPath) {
	child.SetParent(w)
}

// Base type for types having name and path.
type WithNameAndPathBase struct {
	WithNameBase
	WithPathBase
}

func (e *WithNameAndPathBase) Init(path string, name string, separator ...string) {
	e.WithNameBase.Init(name)
	e.WithPathBase.Init(path, separator...)
}

// Base type for types having name and path and children.
type WithNameAndPathParentBase struct {
	WithNameBase
	WithPathParentBase
}

func (e *WithNameAndPathParentBase) Init(path string, name string, separator ...string) {
	e.WithNameBase.Init(name)
	e.WithPathParentBase.Init(path, separator...)
}
