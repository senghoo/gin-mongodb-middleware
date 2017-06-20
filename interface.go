package ginm

type PreCreate interface {
	PreCreate() error
}

type PostCreate interface {
	PostCreate() error
}

type PreDelete interface {
	PreDelete() error
}

type PostDelete interface {
	PostDelete() error
}

type PreUpdate interface {
	PreUpdate() error
}

type PostUpdate interface {
	PostUpdate(pre interface{}) error
}
